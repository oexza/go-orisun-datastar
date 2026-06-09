package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"zombiezen.com/go/sqlite"
)

type AuthUserStore struct {
	db *appdb.DB
}

func NewAuthUserStore(db *appdb.DB) *AuthUserStore {
	return &AuthUserStore{db: db}
}

func (s *AuthUserStore) UserBySessionToken(ctx context.Context, token string) (views.User, error) {
	var row *dbsql.UserBySessionTokenRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserBySessionToken(conn, token)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromSessionRow(row)
}

func (s *AuthUserStore) UserByRegisteredID(ctx context.Context, userRegisteredID string) (views.User, error) {
	var row *dbsql.UserByRegisteredIdRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByRegisteredId(conn, userRegisteredID)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromRegisteredRow(row)
}

func (s *AuthUserStore) UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error) {
	var row *dbsql.UserByIdorRegisteredIdRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByIdorRegisteredId(conn, id)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromIDOrRegisteredRow(row)
}

func (s *AuthUserStore) UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserImage(conn, dbsql.UpdateAuthUserImageParams{
			Image:            stringPtr(imageURL),
			UserRegisteredId: userRegisteredID,
		})
	})
}

func (s *AuthUserStore) MarkEmailVerified(ctx context.Context, userRegisteredID string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceMarkAuthUserEmailVerified(conn, userRegisteredID)
	})
}

func (s *AuthUserStore) userByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
	var row *dbsql.UserByEmailWithPasswordRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByEmailWithPassword(conn, emailAddress)
		return err
	}); err != nil {
		return views.User{}, "", err
	}
	if row == nil || row.Password == nil {
		return views.User{}, "", appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, *row.Password, nil
}
