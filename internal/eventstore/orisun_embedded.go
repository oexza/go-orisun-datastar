package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	orisunconfig "github.com/oexza/Orisun/config"
	embeddedsqlite "github.com/oexza/Orisun/embedded/sqlite"
	orisunlog "github.com/oexza/Orisun/logging"
	orisunapi "github.com/oexza/Orisun/orisun"
)

type EmbeddedOrisun struct {
	store    *embeddedsqlite.Store
	boundary string
}

type EmbeddedConfig struct {
	Boundary     string
	SQLiteDir    string
	NATSStoreDir string
	LogLevel     string
}

func StartEmbeddedOrisun(ctx context.Context, cfg EmbeddedConfig) (*EmbeddedOrisun, error) {
	appConfig, err := orisunconfig.LoadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Boundary == "" {
		cfg.Boundary = "go_orisun_datastar"
	}
	if cfg.SQLiteDir == "" {
		cfg.SQLiteDir = "data/orisun"
	}
	if cfg.NATSStoreDir == "" {
		cfg.NATSStoreDir = "/tmp/go-event-starter-orisun-nats"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	appConfig.Backend.Type = "sqlite"
	appConfig.Sqlite.Dir = cfg.SQLiteDir
	appConfig.Boundaries = fmt.Sprintf(`[{"name":%q,"description":"Starter app events"},{"name":"orisun_admin","description":"Orisun admin boundary"}]`, cfg.Boundary)
	if err := appConfig.ParseBoundaries(); err != nil {
		return nil, err
	}
	appConfig.Admin.Boundary = "orisun_admin"
	appConfig.Nats.StoreDir = cfg.NATSStoreDir
	appConfig.Nats.Port = -1
	appConfig.Nats.Cluster.Enabled = false
	appConfig.Logging.Level = cfg.LogLevel

	logger := orisunlog.InitializeDefaultLogger(appConfig.Logging)
	store, err := embeddedsqlite.Start(ctx, appConfig, logger)
	if err != nil {
		return nil, err
	}
	return &EmbeddedOrisun{store: store, boundary: cfg.Boundary}, nil
}

func (s *EmbeddedOrisun) Close(ctx context.Context) {
	if s != nil && s.store != nil {
		s.store.Close()
	}
}

func (s *EmbeddedOrisun) NATSConnection() *natsgo.Conn {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.NATSConnection()
}

func (s *EmbeddedOrisun) SaveEvents(ctx context.Context, events []DomainEvent, expected Position, scopeEvents []ResolvedEvent, subset Query) (WriteResult, error) {
	toSave := make([]orisunapi.EventWithMapTags, 0, len(events))
	for _, event := range events {
		merged, err := MergeScope(scopeEvents, event)
		if err != nil {
			return WriteResult{}, err
		}
		data := flattenMap(merged.Data)
		data["eventType"] = merged.EventType
		toSave = append(toSave, orisunapi.EventWithMapTags{
			EventId:   merged.EventID,
			EventType: merged.EventType,
			Data:      data,
			Metadata:  merged.Metadata,
		})
	}
	position := toOrisunPosition(expected)
	saved, err := s.store.SaveEvents(ctx, toSave, s.boundary, position, toOrisunQuery(subset))
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Position: fromOrisunPosition(saved)}, nil
}

func (s *EmbeddedOrisun) GetEvents(ctx context.Context, from Position, count int, direction Direction, query Query) ([]ResolvedEvent, error) {
	if count <= 0 {
		count = 100
	}
	req := &orisunapi.GetEventsRequest{
		Boundary:     s.boundary,
		FromPosition: toOrisunPosition(from),
		Count:        uint32(count),
		Direction:    orisunapi.Direction_ASC,
		Query:        toOrisunQuery(query),
	}
	if direction == Backward {
		req.Direction = orisunapi.Direction_DESC
	}
	resp, err := s.store.GetEvents(ctx, req)
	if err != nil {
		return nil, err
	}
	resolved := make([]ResolvedEvent, 0, len(resp.Events))
	for _, event := range resp.Events {
		data := map[string]any{}
		if event.Data != "" {
			if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
				return nil, err
			}
		}
		metadata := map[string]any{}
		if event.Metadata != "" {
			_ = json.Unmarshal([]byte(event.Metadata), &metadata)
		}
		resolved = append(resolved, ResolvedEvent{
			Position: fromOrisunPosition(event.Position),
			Event: DomainEvent{
				EventID:   event.EventId,
				EventType: event.EventType,
				Data:      unflattenMap(data),
				Metadata:  metadata,
			},
		})
	}
	return resolved, nil
}

func (s *EmbeddedOrisun) SubscribeToEvents(ctx context.Context, subscriberName string, after Position, query Query, handle func(context.Context, ResolvedEvent) error) error {
	subscriptionCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := orisunapi.NewMessageHandler[orisunapi.Event](subscriptionCtx)
	errs := make(chan error, 2)

	go func() {
		for {
			event, err := handler.Recv()
			if err != nil {
				errs <- err
				return
			}
			data := map[string]any{}
			if event.Data != "" {
				if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
					errs <- err
					return
				}
			}
			metadata := map[string]any{}
			if event.Metadata != "" {
				_ = json.Unmarshal([]byte(event.Metadata), &metadata)
			}
			if err := handle(subscriptionCtx, ResolvedEvent{
				Position: fromOrisunPosition(event.Position),
				Event:    DomainEvent{EventID: event.EventId, EventType: event.EventType, Data: unflattenMap(data), Metadata: metadata},
			}); err != nil {
				errs <- err
				return
			}
		}
	}()

	go func() {
		errs <- s.store.SubscribeToEvents(subscriptionCtx, s.boundary, subscriberName, toOrisunPosition(after), toOrisunQuery(query), handler)
	}()

	select {
	case err := <-errs:
		cancel()
		handler.Close()
		return err
	case <-ctx.Done():
		cancel()
		handler.Close()
		return ctx.Err()
	}
}

func toOrisunPosition(position Position) *orisunapi.Position {
	return &orisunapi.Position{CommitPosition: position.Commit, PreparePosition: position.Prepare}
}

func fromOrisunPosition(position *orisunapi.Position) Position {
	if position == nil {
		return NoEventPosition
	}
	return Position{Commit: position.CommitPosition, Prepare: position.PreparePosition}
}

func toOrisunQuery(query Query) *orisunapi.Query {
	if len(query.Criteria) == 0 {
		return nil
	}
	criteria := make([]*orisunapi.Criterion, 0, len(query.Criteria))
	for _, criterion := range query.Criteria {
		tags := make([]*orisunapi.Tag, 0, len(criterion.Tags))
		for _, tag := range criterion.Tags {
			tags = append(tags, &orisunapi.Tag{Key: tag.Key, Value: tag.Value})
		}
		criteria = append(criteria, &orisunapi.Criterion{Tags: tags})
	}
	return &orisunapi.Query{Criteria: criteria}
}

func flattenMap(input map[string]any) map[string]any {
	output := map[string]any{}
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				next := key
				if prefix != "" {
					next = prefix + "." + key
				}
				walk(next, child)
			}
		default:
			output[prefix] = typed
		}
	}
	for key, value := range input {
		walk(key, value)
	}
	return output
}

func unflattenMap(input map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range input {
		parts := strings.Split(key, ".")
		current := output
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = value
				continue
			}
			next, ok := current[part].(map[string]any)
			if !ok {
				next = map[string]any{}
				current[part] = next
			}
			current = next
		}
	}
	return output
}

func PollingStoreDir() string {
	return fmt.Sprintf("/tmp/go-event-starter-orisun-%d", time.Now().UnixNano())
}
