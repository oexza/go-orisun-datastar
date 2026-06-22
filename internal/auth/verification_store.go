package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oexza/go-orisun-datastar/internal/dbsql"
)

type VerificationStore struct {
	queries *dbsql.Queries
}

func NewVerificationStore(db *pgxpool.Pool) *VerificationStore {
	return &VerificationStore{queries: dbsql.New(db)}
}

func (s *VerificationStore) CreateEmailVerificationOTP(ctx context.Context, userRegisteredID, otpID, code string, expiresAt time.Time) error {
	return s.queries.CreateAuthVerification(ctx, dbsql.CreateAuthVerificationParams{
		ID:         otpID,
		Identifier: "email:" + userRegisteredID,
		Value:      code,
		ExpiresAt:  pgTime(expiresAt),
	})
}

func (s *VerificationStore) CreatePasswordReset(ctx context.Context, userRegisteredID, requestID, token string, expiresAt time.Time) error {
	return s.queries.CreateAuthVerification(ctx, dbsql.CreateAuthVerificationParams{
		ID:         requestID,
		Identifier: "password-reset:" + userRegisteredID,
		Value:      token,
		ExpiresAt:  pgTime(expiresAt),
	})
}

func (s *VerificationStore) PasswordResetByToken(ctx context.Context, token string) (PasswordResetVerification, error) {
	verification, err := s.queries.PasswordResetVerificationByToken(ctx, token)
	if err != nil {
		return PasswordResetVerification{}, errors.New("invalid or expired reset token")
	}
	return PasswordResetVerification{
		ID:     verification.ID,
		UserID: strings.TrimPrefix(verification.Identifier, "password-reset:"),
	}, nil
}

type PasswordResetVerification struct {
	ID     string
	UserID string
}

func pgTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
