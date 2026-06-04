package profile

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/hono-event-starter-go/internal/eventstore"
)

const (
	userRegistered  = "UserRegistered"
	userNameChanged = "UserNameChanged"
)

type ReadModel struct {
	db *pgxpool.Pool
}

func NewReadModel(db *pgxpool.Pool) *ReadModel {
	return &ReadModel{db: db}
}

func (m *ReadModel) UpsertRegisteredUser(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	data := resolved.Event.Data
	userRegisteredID, _ := data["userRegisteredId"].(string)
	username, _ := data["username"].(string)
	emailAddress, _ := data["email"].(string)
	firstName, _ := data["firstName"].(string)
	lastName, _ := data["lastName"].(string)
	name := strings.TrimSpace(firstName + " " + lastName)
	_, err := m.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, name, username, email, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
		    name = EXCLUDED.name,
		    username = EXCLUDED.username,
		    email = EXCLUDED.email,
		    last_event_commit_position = EXCLUDED.last_event_commit_position,
		    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
		    updated_at = now()
	`, userRegisteredID, name, username, emailAddress, resolved.Position.Commit, resolved.Position.Prepare)
	return err
}

func (m *ReadModel) UpdateName(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	name, _ := resolved.Event.Data["name"].(string)
	_, err := m.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, name, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
		    name = EXCLUDED.name,
		    last_event_commit_position = EXCLUDED.last_event_commit_position,
		    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
		    updated_at = now()
	`, userRegisteredID, name, resolved.Position.Commit, resolved.Position.Prepare)
	return err
}

func (m *ReadModel) UpdateBio(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	bio, _ := resolved.Event.Data["bio"].(string)
	_, err := m.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, bio, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
		    bio = EXCLUDED.bio,
		    last_event_commit_position = EXCLUDED.last_event_commit_position,
		    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
		    updated_at = now()
	`, userRegisteredID, bio, resolved.Position.Commit, resolved.Position.Prepare)
	return err
}

func (m *ReadModel) UpdateImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	_, err := m.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, image, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
		    image = EXCLUDED.image,
		    last_event_commit_position = EXCLUDED.last_event_commit_position,
		    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
		    updated_at = now()
	`, userRegisteredID, url, resolved.Position.Commit, resolved.Position.Prepare)
	return err
}

func (m *ReadModel) UpdateHeaderImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	_, err := m.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, header_image_url, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
		    header_image_url = EXCLUDED.header_image_url,
		    last_event_commit_position = EXCLUDED.last_event_commit_position,
		    last_event_prepare_position = EXCLUDED.last_event_prepare_position,
		    updated_at = now()
	`, userRegisteredID, url, resolved.Position.Commit, resolved.Position.Prepare)
	return err
}

type ReadModelEventHandler struct {
	global    *eventstore.GlobalEventHandler
	readModel *ReadModel
}

func NewReadModelEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, readModel *ReadModel, logger *slog.Logger) (*ReadModelEventHandler, error) {
	handler := &ReadModelEventHandler{readModel: readModel}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            "profile_read_model_event_handler",
		Query:           readModelEventHandlerQuery(),
		Logger:          logger,
		MaxEventRetries: -1,
		HandleEvent:     handler.handle,
	})
	if err != nil {
		return nil, err
	}
	handler.global = global
	return handler, nil
}

func (h *ReadModelEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *ReadModelEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *ReadModelEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	switch resolved.Event.EventType {
	case userRegistered:
		return h.readModel.UpsertRegisteredUser(ctx, resolved)
	case userNameChanged:
		return h.readModel.UpdateName(ctx, resolved)
	case ProfileBioUpdated:
		return h.readModel.UpdateBio(ctx, resolved)
	case ProfileImageUploaded:
		return h.readModel.UpdateImage(ctx, resolved)
	case ProfileHeaderImageUploaded:
		return h.readModel.UpdateHeaderImage(ctx, resolved)
	default:
		return nil
	}
}

func readModelEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: userRegistered}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: userNameChanged}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileBioUpdated}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileImageUploaded}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileHeaderImageUploaded}}},
	}}
}
