package profile

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
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

func stringPtr(value string) *string {
	return &value
}
