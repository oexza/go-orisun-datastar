package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/protectedpii"
	"zombiezen.com/go/sqlite"
)

type SubjectPiiKeyPort interface {
	GetOrCreateSubjectDataKey(ctx context.Context, subjectID string) (protectedpii.SubjectDataKey, error)
	GetSubjectDataKey(ctx context.Context, subjectID string) (protectedpii.SubjectDataKey, bool, error)
	DestroySubjectKey(ctx context.Context, subjectID string) error
}

type SubjectPiiKeyStore struct {
	db        *appdb.DB
	protector *protectedpii.Protector
}

func NewSubjectPiiKeyStore(db *appdb.DB, protector *protectedpii.Protector) *SubjectPiiKeyStore {
	return &SubjectPiiKeyStore{db: db, protector: protector}
}

func (s *SubjectPiiKeyStore) GetOrCreateSubjectDataKey(ctx context.Context, subjectID string) (protectedpii.SubjectDataKey, error) {
	if key, ok, err := s.GetSubjectDataKey(ctx, subjectID); err != nil || ok {
		return key, err
	}
	generated, err := protectedpii.GenerateSubjectDataKey()
	if err != nil {
		return protectedpii.SubjectDataKey{}, err
	}
	protected, err := s.protector.ProtectSubjectDataKey(generated)
	if err != nil {
		return protectedpii.SubjectDataKey{}, err
	}
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateSubjectPiiKey(conn, dbsql.CreateSubjectPiiKeyParams{
			SubjectId:        subjectID,
			EncryptedDataKey: protected.Ciphertext,
			EncryptionNonce:  protected.Nonce,
			KeyVersion:       protected.KeyID,
		})
	}); err != nil {
		return protectedpii.SubjectDataKey{}, err
	}
	if key, ok, err := s.GetSubjectDataKey(ctx, subjectID); err != nil || ok {
		return key, err
	}
	return generated, nil
}

func (s *SubjectPiiKeyStore) GetSubjectDataKey(ctx context.Context, subjectID string) (protectedpii.SubjectDataKey, bool, error) {
	var row *dbsql.SubjectPiiKeyRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceSubjectPiiKey(conn, subjectID)
		if err == appdb.ErrNoRows {
			return nil
		}
		return err
	}); err != nil {
		return protectedpii.SubjectDataKey{}, false, err
	}
	if row == nil {
		return protectedpii.SubjectDataKey{}, false, nil
	}
	key, err := s.protector.UnprotectSubjectDataKey(protectedpii.Value{
		Version:    1,
		KeyID:      row.KeyVersion,
		Nonce:      row.EncryptionNonce,
		Ciphertext: row.EncryptedDataKey,
	})
	if err != nil {
		return protectedpii.SubjectDataKey{}, false, err
	}
	return key, true, nil
}

func (s *SubjectPiiKeyStore) DestroySubjectKey(ctx context.Context, subjectID string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceDeleteSubjectPiiKey(conn, subjectID)
	})
}
