package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"zombiezen.com/go/sqlite"
)

type AccountDeletionStorage interface {
	DeleteObject(ctx context.Context, key string) error
	ObjectKeyFromPublicURL(url string) (string, bool)
}

type AccountDataDeletionStore struct {
	db      *appdb.DB
	storage AccountDeletionStorage
}

func NewAccountDataDeletionStore(db *appdb.DB, storage AccountDeletionStorage) *AccountDataDeletionStore {
	return &AccountDataDeletionStore{db: db, storage: storage}
}

func (s *AccountDataDeletionStore) DeleteAccountData(ctx context.Context, userRegisteredID string) error {
	if err := s.deleteKnownMedia(ctx, userRegisteredID); err != nil {
		return err
	}
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		if err := dbsql.OnceDeleteAuthVerificationsByRegisteredId(conn, stringPtr(userRegisteredID)); err != nil {
			return err
		}
		if err := dbsql.OnceDeleteAuthSessionsByRegisteredId(conn, userRegisteredID); err != nil {
			return err
		}
		if err := dbsql.OnceDeleteAuthAccountsByRegisteredId(conn, userRegisteredID); err != nil {
			return err
		}
		if err := dbsql.OnceDeleteProfileByRegisteredId(conn, userRegisteredID); err != nil {
			return err
		}
		if err := dbsql.OnceDeleteTodosByRegisteredId(conn, userRegisteredID); err != nil {
			return err
		}
		return dbsql.OnceDeleteAuthUserByRegisteredId(conn, userRegisteredID)
	})
}

func (s *AccountDataDeletionStore) deleteKnownMedia(ctx context.Context, userRegisteredID string) error {
	if s.storage == nil {
		return nil
	}
	var row *dbsql.ProfileUserRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceProfileUser(conn, userRegisteredID)
		if err == appdb.ErrNoRows {
			return nil
		}
		return err
	}); err != nil {
		return err
	}
	if row == nil {
		return nil
	}
	for _, url := range []string{row.Image, row.HeaderImageUrl} {
		if url == "" {
			continue
		}
		key, ok := s.storage.ObjectKeyFromPublicURL(url)
		if !ok {
			continue
		}
		if err := s.storage.DeleteObject(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
