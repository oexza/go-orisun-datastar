package eventstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/OrisunLabs/Orisun/config"
	orisunlog "github.com/OrisunLabs/Orisun/logging"
	natsruntime "github.com/OrisunLabs/Orisun/nats"
	orisunapi "github.com/OrisunLabs/Orisun/orisun"
	postgresbackend "github.com/OrisunLabs/Orisun/postgres"
	natsgo "github.com/nats-io/nats.go"
)

type EmbeddedOrisun struct {
	store        *orisunapi.OrisunServer
	retriever    orisunapi.EventsRetriever
	indexManager orisunapi.BoundaryIndexManager
	cancel       context.CancelFunc
	natsRuntime  *natsruntime.Runtime
	closePG      func(context.Context)
	boundary     string
}

type EmbeddedConfig struct {
	Boundary         string
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDatabase string
	PostgresSSLMode  string
	NATSStoreDir     string
	LogLevel         string
}

func StartEmbeddedOrisun(ctx context.Context, cfg EmbeddedConfig) (*EmbeddedOrisun, error) {
	appConfig, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Boundary == "" {
		cfg.Boundary = "go_orisun_datastar"
	}
	if cfg.PostgresSSLMode == "" {
		cfg.PostgresSSLMode = "disable"
	}
	if cfg.NATSStoreDir == "" {
		cfg.NATSStoreDir = "/tmp/go-event-starter-orisun-nats"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	appConfig.Backend.Type = "postgres"
	appConfig.Postgres.Host = cfg.PostgresHost
	appConfig.Postgres.Port = cfg.PostgresPort
	appConfig.Postgres.User = cfg.PostgresUser
	appConfig.Postgres.Password = cfg.PostgresPassword
	appConfig.Postgres.Name = cfg.PostgresDatabase
	appConfig.Postgres.SSLMode = cfg.PostgresSSLMode
	appConfig.Postgres.Schemas = cfg.Boundary + ":public,orisun_admin:admin"
	appConfig.Postgres.ListenEnabled = true
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
	runCtx, cancel := context.WithCancel(ctx)

	natsRuntime, err := natsruntime.Start(runCtx, appConfig.Nats, logger)
	if err != nil {
		cancel()
		return nil, err
	}
	saveEvents, getEvents, lockProvider, adminDB, eventPublishing, pgListener := postgresbackend.InitializePostgresDatabase(runCtx, appConfig.Postgres, appConfig.Admin, natsRuntime.JetStream, logger)
	store, err := orisunapi.NewOrisunServer(runCtx, saveEvents, getEvents, lockProvider, natsRuntime.JetStream, appConfig.GetBoundaryNames(), logger)
	if err != nil {
		cancel()
		natsRuntime.Close()
		return nil, err
	}

	var signalProvider func(string) orisunapi.EventSignal
	var closePG func(context.Context)
	if pgListener != nil {
		listenerCtx, stopListener := context.WithCancel(runCtx)
		go pgListener.Start(listenerCtx)
		signalProvider = func(boundary string) orisunapi.EventSignal {
			return pgListener.Signal(boundary, 30*time.Second)
		}
		closePG = func(ctx context.Context) {
			stopListener()
			waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Second)
			defer waitCancel()
			pgListener.Close(waitCtx)
		}
	} else {
		signalProvider = func(boundary string) orisunapi.EventSignal {
			return orisunapi.NewPollingSignal(time.Second)
		}
	}
	orisunapi.StartEventPolling(runCtx, appConfig, lockProvider, getEvents, natsRuntime.JetStream, eventPublishing, signalProvider, logger)

	return &EmbeddedOrisun{
		store:        store,
		retriever:    getEvents,
		indexManager: adminDB,
		cancel:       cancel,
		natsRuntime:  natsRuntime,
		closePG:      closePG,
		boundary:     cfg.Boundary,
	}, nil
}

func (s *EmbeddedOrisun) Close(ctx context.Context) {
	if s == nil {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.closePG != nil {
		s.closePG(ctx)
	}
	if s.natsRuntime != nil {
		s.natsRuntime.Close()
	}
}

func (s *EmbeddedOrisun) NATSConnection() *natsgo.Conn {
	if s == nil || s.natsRuntime == nil {
		return nil
	}
	return s.natsRuntime.Conn
}

func (s *EmbeddedOrisun) SaveEvents(ctx context.Context, events []DomainEvent, expected Position, scopeEvents []ResolvedEvent, subset Query) (WriteResult, error) {
	toSave := make([]orisunapi.EventWithMapTags, 0, len(events))
	for _, event := range events {
		event = withScopeEventIDs(event, scopeEvents)
		data := flattenMap(event.Data)
		toSave = append(toSave, orisunapi.EventWithMapTags{
			EventId:   event.EventID,
			EventType: event.EventType,
			Data:      data,
			Metadata:  event.Metadata,
		})
	}
	position := toOrisunPosition(expected)
	saved, err := s.store.SaveEvents(ctx, toSave, s.boundary, position, toOrisunQuery(subset))
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Position: fromOrisunPosition(saved)}, nil
}

func withScopeEventIDs(event DomainEvent, scopeEvents []ResolvedEvent) DomainEvent {
	sortedEvents := append([]ResolvedEvent(nil), scopeEvents...)
	sort.SliceStable(sortedEvents, func(i, j int) bool {
		left := sortedEvents[i].Position
		right := sortedEvents[j].Position
		if left.Commit != right.Commit {
			return left.Commit < right.Commit
		}
		return left.Prepare < right.Prepare
	})

	ids := make([]string, 0, len(sortedEvents))
	seen := make(map[string]struct{}, len(sortedEvents))
	for _, resolved := range sortedEvents {
		id := resolved.Event.EventID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	metadata := make(map[string]any, len(event.Metadata)+1)
	for key, value := range event.Metadata {
		metadata[key] = value
	}
	metadata["scope_event_ids"] = ids
	event.Metadata = metadata
	return event
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
	resolved := make([]ResolvedEvent, 0, len(resp))
	for _, event := range resp {
		mapped, err := fromOrisunReadEvent(event)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, mapped)
	}
	return resolved, nil
}

func (s *EmbeddedOrisun) GetLatestByCriteria(ctx context.Context, criteria []Criterion) (LatestByCriteriaResult, error) {
	batch, err := s.retriever.GetLatestByCriteria(ctx, orisunapi.LatestByCriteriaQuery{
		Boundary: s.boundary,
		Criteria: toOrisunReadCriteria(criteria),
	})
	if err != nil {
		return LatestByCriteriaResult{}, err
	}
	result := LatestByCriteriaResult{
		Results: make([]LatestCriterionResult, 0, len(batch.Matches)),
		ContextPosition: Position{
			Commit:  batch.ContextCommitPosition,
			Prepare: batch.ContextPreparePosition,
		},
	}
	for i, latest := range batch.Matches {
		mapped := LatestCriterionResult{Criterion: criteria[i]}
		if latest.Found {
			event, err := fromOrisunReadEvent(latest.Event)
			if err != nil {
				return LatestByCriteriaResult{}, err
			}
			mapped.Event = &event
		}
		result.Results = append(result.Results, mapped)
	}
	return result, nil
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

func fromOrisunReadEvent(event orisunapi.ReadEvent) (ResolvedEvent, error) {
	data := map[string]any{}
	if event.Data != "" {
		if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
			return ResolvedEvent{}, err
		}
	}
	metadata := map[string]any{}
	if event.Metadata != "" {
		_ = json.Unmarshal([]byte(event.Metadata), &metadata)
	}
	return ResolvedEvent{
		Position: Position{Commit: event.CommitPosition, Prepare: event.PreparePosition},
		Event: DomainEvent{
			EventID:   event.EventId,
			EventType: event.EventType,
			Data:      unflattenMap(data),
			Metadata:  metadata,
		},
	}, nil
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
	return &orisunapi.Query{Criteria: toOrisunCriteria(query.Criteria)}
}

func toOrisunCriteria(input []Criterion) []*orisunapi.Criterion {
	criteria := make([]*orisunapi.Criterion, 0, len(input))
	for _, criterion := range input {
		tags := make([]*orisunapi.Tag, 0, len(criterion.Tags))
		for _, tag := range criterion.Tags {
			tags = append(tags, &orisunapi.Tag{Key: tag.Key, Value: tag.Value})
		}
		criteria = append(criteria, &orisunapi.Criterion{Tags: tags})
	}
	return criteria
}

func toOrisunReadCriteria(input []Criterion) []orisunapi.ReadCriterion {
	criteria := make([]orisunapi.ReadCriterion, 0, len(input))
	for _, criterion := range input {
		tags := make([]orisunapi.ReadTag, 0, len(criterion.Tags))
		for _, tag := range criterion.Tags {
			tags = append(tags, orisunapi.ReadTag{Key: tag.Key, Value: tag.Value})
		}
		criteria = append(criteria, orisunapi.ReadCriterion{Tags: tags})
	}
	return criteria
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
