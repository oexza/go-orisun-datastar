package profile

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oexza/go-orisun-datastar/internal/auth"

	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/protectedpii"
	"github.com/oexza/go-orisun-datastar/internal/views"
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

func (m *ReadModel) User(ctx context.Context, userRegisteredID string) (views.User, error) {
	row, err := m.queries.ProfileUser(ctx, userRegisteredID)
	if err != nil {
		return views.User{}, err
	}
	return views.User{
		ID:               row.UserID,
		UserRegisteredID: row.UserID,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func (m *ReadModel) UpsertRegisteredUser(ctx context.Context, resolved eventstore.ResolvedEvent, keys auth.SubjectPiiKeyPort) error {
	data := resolved.Event.Data
	userRegisteredID, _ := data["userRegisteredId"].(string)
	protector := protectedpii.FromEnv()
	subjectKey, ok, err := keys.GetSubjectDataKey(ctx, userRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	username := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, data, "username")
	emailAddress := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, data, "email")
	firstName := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, data, "firstName")
	lastName := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, data, "lastName")
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

func (m *ReadModel) UpdateName(ctx context.Context, resolved eventstore.ResolvedEvent, keys auth.SubjectPiiKeyPort) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	subjectKey, ok, err := keys.GetSubjectDataKey(ctx, userRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	name := protectedpii.MustDecryptEventStringWithDataKey(protectedpii.FromEnv(), subjectKey, resolved.Event.Data, "name")
	return m.queries.UpsertProfileName(ctx, dbsql.UpsertProfileNameParams{
		UserID:                   userRegisteredID,
		Name:                     stringPtr(name),
		LastEventCommitPosition:  resolved.Position.Commit,
		LastEventPreparePosition: resolved.Position.Prepare,
	})
}

func (m *ReadModel) UpdateBio(ctx context.Context, resolved eventstore.ResolvedEvent, keys auth.SubjectPiiKeyPort) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	subjectKey, ok, err := keys.GetSubjectDataKey(ctx, userRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	bio := protectedpii.MustDecryptEventStringWithDataKey(protectedpii.FromEnv(), subjectKey, resolved.Event.Data, "bio")
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
