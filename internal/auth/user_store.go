package auth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type AuthUserStore struct {
	queries *dbsql.Queries
}

func NewAuthUserStore(db *pgxpool.Pool) *AuthUserStore {
	return &AuthUserStore{queries: dbsql.New(db)}
}

func (s *AuthUserStore) UserBySessionToken(ctx context.Context, token string) (views.User, error) {
	row, err := s.queries.UserBySessionToken(ctx, token)
	if err != nil {
		return views.User{}, err
	}
	return userFromSessionRow(row), nil
}

func (s *AuthUserStore) UserByRegisteredID(ctx context.Context, userRegisteredID string) (views.User, error) {
	row, err := s.queries.UserByRegisteredID(ctx, userRegisteredID)
	if err != nil {
		return views.User{}, err
	}
	return userFromRegisteredRow(row), nil
}

func (s *AuthUserStore) UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error) {
	row, err := s.queries.UserByIDOrRegisteredID(ctx, id)
	if err != nil {
		return views.User{}, err
	}
	return userFromIDOrRegisteredRow(row), nil
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

func (s *AuthUserStore) userByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
	row, err := s.queries.UserByEmailWithPassword(ctx, emailAddress)
	if err != nil {
		return views.User{}, "", err
	}
	if row.Password == nil {
		return views.User{}, "", pgx.ErrNoRows
	}
	return views.User{
		ID:               row.ID,
		UserRegisteredID: row.UserRegisteredID,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, *row.Password, nil
}
