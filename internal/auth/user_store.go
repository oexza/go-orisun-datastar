package auth

import (
	"context"
	"strings"

	"github.com/OrisunLabs/go-orisun-datastar/internal/appdb"
	"github.com/OrisunLabs/go-orisun-datastar/internal/dbsql"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
	"zombiezen.com/go/sqlite"
)

type AuthUserStore struct {
	db *appdb.DB
}

func NewAuthUserStore(db *appdb.DB) *AuthUserStore {
	return &AuthUserStore{db: db}
}

func (s *AuthUserStore) CreateRegisteredUserAccount(ctx context.Context, registered RegisterUserResult) error {
	userID := registered.UserRegisteredID
	name := strings.TrimSpace(registered.FirstName + " " + registered.LastName)
	if name == "" {
		name = registered.Username
	}
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		if err := dbsql.OnceCreateAuthUser(conn, dbsql.CreateAuthUserParams{
			Id:               userID,
			Name:             name,
			Email:            registered.Email,
			Username:         stringPtr(registered.Username),
			UserRegisteredId: registered.UserRegisteredID,
		}); err != nil {
			return err
		}
		if registered.PasswordHash == "" {
			return nil
		}
		user, err := dbsql.OnceUserByRegisteredId(conn, registered.UserRegisteredID)
		if err != nil {
			return err
		}
		if user == nil {
			return appdb.ErrNoRows
		}
		return dbsql.OnceCreateAuthAccount(conn, dbsql.CreateAuthAccountParams{
			Id:        registered.UserRegisteredID + ":credential",
			AccountId: registered.Email,
			UserId:    user.Id,
			Password:  stringPtr(registered.PasswordHash),
		})
	}); err != nil {
		return err
	}
	return nil
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

func (s *AuthUserStore) UpdateName(ctx context.Context, userID, name string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserName(conn, dbsql.UpdateAuthUserNameParams{Name: name, Id: userID})
	})
}

func (s *AuthUserStore) UpdateNameByRegisteredID(ctx context.Context, userRegisteredID, name string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserNameByRegisteredId(conn, dbsql.UpdateAuthUserNameByRegisteredIdParams{Name: name, UserRegisteredId: userRegisteredID})
	})
}

func (s *AuthUserStore) UpdatePasswordByRegisteredID(ctx context.Context, userRegisteredID, passwordHash string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthAccountPasswordByRegisteredId(conn, dbsql.UpdateAuthAccountPasswordByRegisteredIdParams{
			Password:         stringPtr(passwordHash),
			UserRegisteredId: userRegisteredID,
		})
	})
}

func (s *AuthUserStore) UserByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
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
