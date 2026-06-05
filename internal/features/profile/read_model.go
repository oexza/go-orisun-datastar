package profile

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/hono-event-starter-go/internal/dbsql"
	"github.com/example/hono-event-starter-go/internal/eventstore"
)

const (
	userRegistered  = "UserRegistered"
	userNameChanged = "UserNameChanged"
)

type ReadModel struct {
	queries *dbsql.Queries
}

func NewReadModel(db *pgxpool.Pool) *ReadModel {
	return &ReadModel{queries: dbsql.New(db)}
}

func (m *ReadModel) UpsertRegisteredUser(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	data := resolved.Event.Data
	userRegisteredID, _ := data["userRegisteredId"].(string)
	username, _ := data["username"].(string)
	emailAddress, _ := data["email"].(string)
	firstName, _ := data["firstName"].(string)
	lastName, _ := data["lastName"].(string)
	name := strings.TrimSpace(firstName + " " + lastName)
	return m.queries.UpsertRegisteredProfileUser(ctx, dbsql.UpsertRegisteredProfileUserParams{
		UserID:                   userRegisteredID,
		Name:                     stringPtr(name),
		Username:                 stringPtr(username),
		Email:                    stringPtr(emailAddress),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
}

func (m *ReadModel) UpdateName(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	name, _ := resolved.Event.Data["name"].(string)
	return m.queries.UpsertProfileName(ctx, dbsql.UpsertProfileNameParams{
		UserID:                   userRegisteredID,
		Name:                     stringPtr(name),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
}

func (m *ReadModel) UpdateBio(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	bio, _ := resolved.Event.Data["bio"].(string)
	return m.queries.UpsertProfileBio(ctx, dbsql.UpsertProfileBioParams{
		UserID:                   userRegisteredID,
		Bio:                      stringPtr(bio),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
}

func (m *ReadModel) UpdateImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	return m.queries.UpsertProfileImage(ctx, dbsql.UpsertProfileImageParams{
		UserID:                   userRegisteredID,
		Image:                    stringPtr(url),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
}

func (m *ReadModel) UpdateHeaderImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	return m.queries.UpsertProfileHeaderImage(ctx, dbsql.UpsertProfileHeaderImageParams{
		UserID:                   userRegisteredID,
		HeaderImageUrl:           stringPtr(url),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
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

func stringPtr(value string) *string {
	return &value
}
