package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"zombiezen.com/go/sqlite"
)

type VerificationStore struct {
	db *appdb.DB
}

func NewVerificationStore(db *appdb.DB) *VerificationStore {
	return &VerificationStore{db: db}
}

func (s *VerificationStore) CreateEmailVerificationOTP(ctx context.Context, userRegisteredID, otpID, code string, expiresAt time.Time) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         otpID,
			Identifier: "email:" + userRegisteredID,
			Value:      code,
			ExpiresAt:  appdb.SQLTime(expiresAt),
		})
	})
}

func (s *VerificationStore) CreatePasswordReset(ctx context.Context, userRegisteredID, requestID, token string, expiresAt time.Time) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         requestID,
			Identifier: "password-reset:" + userRegisteredID,
			Value:      token,
			ExpiresAt:  appdb.SQLTime(expiresAt),
		})
	})
}

func (s *VerificationStore) PasswordResetByToken(ctx context.Context, token string) (PasswordResetVerification, error) {
	var verification *dbsql.PasswordResetVerificationByTokenRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		verification, err = dbsql.OncePasswordResetVerificationByToken(conn, token)
		return err
	}); err != nil || verification == nil {
		return PasswordResetVerification{}, errors.New("invalid or expired reset token")
	}
	return PasswordResetVerification{
		ID:     verification.Id,
		UserID: strings.TrimPrefix(verification.Identifier, "password-reset:"),
	}, nil
}

type PasswordResetVerification struct {
	ID     string
	UserID string
}
