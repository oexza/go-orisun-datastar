package profile

import (
	"context"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"zombiezen.com/go/sqlite"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const (
	userRegistered  = "UserRegistered"
	userNameChanged = "UserNameChanged"
)

type ReadModel struct {
	db *appdb.DB
}

func NewReadModel(db *appdb.DB) *ReadModel {
	return &ReadModel{db: db}
}

func (m *ReadModel) User(ctx context.Context, userRegisteredID string) (views.User, error) {
	var row *dbsql.ProfileUserRes
	err := m.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceProfileUser(conn, userRegisteredID)
		return err
	})
	if err != nil {
		return views.User{}, err
	}
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.UserId,
		UserRegisteredID: row.UserId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func (m *ReadModel) UpsertRegisteredUser(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	data := resolved.Event.Data
	userRegisteredID, _ := data["userRegisteredId"].(string)
	username, _ := data["username"].(string)
	emailAddress, _ := data["email"].(string)
	firstName, _ := data["firstName"].(string)
	lastName, _ := data["lastName"].(string)
	name := strings.TrimSpace(firstName + " " + lastName)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertRegisteredProfileUser(conn, dbsql.UpsertRegisteredProfileUserParams{
			UserId:                   userRegisteredID,
			Name:                     stringPtr(name),
			Username:                 stringPtr(username),
			Email:                    stringPtr(emailAddress),
			LastEventCommitPosition:  resolved.Position.Commit,
			LastEventPreparePosition: resolved.Position.Prepare,
		})
	})
}

func (m *ReadModel) UpdateName(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	name, _ := resolved.Event.Data["name"].(string)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertProfileName(conn, dbsql.UpsertProfileNameParams{
			UserId:                   userRegisteredID,
			Name:                     stringPtr(name),
			LastEventCommitPosition:  resolved.Position.Commit,
			LastEventPreparePosition: resolved.Position.Prepare,
		})
	})
}

func (m *ReadModel) UpdateBio(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	bio, _ := resolved.Event.Data["bio"].(string)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertProfileBio(conn, dbsql.UpsertProfileBioParams{
			UserId:                   userRegisteredID,
			Bio:                      stringPtr(bio),
			LastEventCommitPosition:  resolved.Position.Commit,
			LastEventPreparePosition: resolved.Position.Prepare,
		})
	})
}

func (m *ReadModel) UpdateImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertProfileImage(conn, dbsql.UpsertProfileImageParams{
			UserId:                   userRegisteredID,
			Image:                    stringPtr(url),
			LastEventCommitPosition:  resolved.Position.Commit,
			LastEventPreparePosition: resolved.Position.Prepare,
		})
	})
}

func (m *ReadModel) UpdateHeaderImage(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	url, _ := resolved.Event.Data["imageUrl"].(string)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertProfileHeaderImage(conn, dbsql.UpsertProfileHeaderImageParams{
			UserId:                   userRegisteredID,
			HeaderImageUrl:           stringPtr(url),
			LastEventCommitPosition:  resolved.Position.Commit,
			LastEventPreparePosition: resolved.Position.Prepare,
		})
	})
}

func stringPtr(value string) *string {
	return &value
}
