package auth

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OrisunLabs/go-orisun-datastar/internal/dbsql"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
)

type AuthUserStore struct {
	queries *dbsql.Queries
}

func NewAuthUserStore(db *pgxpool.Pool) *AuthUserStore {
	return &AuthUserStore{queries: dbsql.New(db)}
}

func (s *AuthUserStore) CreateRegisteredUserAccount(ctx context.Context, registered RegisterUserResult) error {
	userID := registered.UserRegisteredID
	name := strings.TrimSpace(registered.FirstName + " " + registered.LastName)
	if name == "" {
		name = registered.Username
	}
	if err := s.queries.CreateAuthUser(ctx, dbsql.CreateAuthUserParams{
		ID:               userID,
		Name:             name,
		Email:            registered.Email,
		Username:         stringPtr(registered.Username),
		UserRegisteredID: registered.UserRegisteredID,
	}); err != nil {
		return err
	}
	if registered.PasswordHash == "" {
		return nil
	}
	return s.queries.CreateAuthAccount(ctx, dbsql.CreateAuthAccountParams{
		ID:        registered.UserRegisteredID + ":credential",
		AccountID: registered.Email,
		UserID:    userID,
		Password:  stringPtr(registered.PasswordHash),
	})
}

func (s *AuthUserStore) UserBySessionToken(ctx context.Context, token string) (views.User, error) {
	row, err := s.queries.UserBySessionToken(ctx, token)
	if err != nil {
		return views.User{}, err
	}
	return userFromSessionRow(row)
}

func (s *AuthUserStore) UserByRegisteredID(ctx context.Context, userRegisteredID string) (views.User, error) {
	row, err := s.queries.UserByRegisteredID(ctx, userRegisteredID)
	if err != nil {
		return views.User{}, err
	}
	return userFromRegisteredRow(row)
}

func (s *AuthUserStore) UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error) {
	row, err := s.queries.UserByIDOrRegisteredID(ctx, id)
	if err != nil {
		return views.User{}, err
	}
	return userFromIDOrRegisteredRow(row)
}

func (s *AuthUserStore) UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error {
	return s.queries.UpdateAuthUserImage(ctx, dbsql.UpdateAuthUserImageParams{
		Image:            stringPtr(imageURL),
		UserRegisteredID: userRegisteredID,
	})
}

func (s *AuthUserStore) MarkEmailVerified(ctx context.Context, userRegisteredID string) error {
	return s.queries.MarkAuthUserEmailVerified(ctx, userRegisteredID)
}

func (s *AuthUserStore) UpdateName(ctx context.Context, userID, name string) error {
	return s.queries.UpdateAuthUserName(ctx, dbsql.UpdateAuthUserNameParams{Name: name, ID: userID})
}

func (s *AuthUserStore) UpdateNameByRegisteredID(ctx context.Context, userRegisteredID, name string) error {
	return s.queries.UpdateAuthUserNameByRegisteredID(ctx, dbsql.UpdateAuthUserNameByRegisteredIDParams{
		Name:             name,
		UserRegisteredID: userRegisteredID,
	})
}

func (s *AuthUserStore) UpdatePasswordByRegisteredID(ctx context.Context, userRegisteredID, passwordHash string) error {
	return s.queries.UpdateAuthAccountPasswordByRegisteredID(ctx, dbsql.UpdateAuthAccountPasswordByRegisteredIDParams{
		Password:         stringPtr(passwordHash),
		UserRegisteredID: userRegisteredID,
	})
}

func (s *AuthUserStore) UserByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
	row, err := s.queries.UserByEmailWithPassword(ctx, emailAddress)
	if err != nil {
		return views.User{}, "", err
	}
	return userFromEmailWithPasswordRow(row)
}
