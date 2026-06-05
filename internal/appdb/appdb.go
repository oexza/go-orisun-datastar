package appdb

import (
	"context"
	"errors"
	"time"

	toolbeltdb "github.com/delaneyj/toolbelt/db"
	appmigrations "github.com/oexza/go-orisun-datastar/migrations"
)

var ErrNoRows = errors.New("no rows in result set")

type DB struct {
	inner *toolbeltdb.Database
}

type TxFn = toolbeltdb.TxFn

func Open(ctx context.Context, filename string) (*DB, error) {
	migrations, err := appmigrations.SQL()
	if err != nil {
		return nil, err
	}
	inner, err := toolbeltdb.NewDatabase(ctx,
		toolbeltdb.DatabaseWithFilename(filename),
		toolbeltdb.DatabaseWithMigrations(migrations),
		toolbeltdb.DatabaseWithPragmas("foreign_keys = ON", "busy_timeout = 5000"),
	)
	if err != nil {
		return nil, err
	}
	return &DB{inner: inner}, nil
}

func (db *DB) Close() error {
	if db == nil || db.inner == nil {
		return nil
	}
	return db.inner.Close()
}

func (db *DB) ReadTX(ctx context.Context, fn TxFn) error {
	return db.inner.ReadTX(ctx, fn)
}

func (db *DB) WriteTX(ctx context.Context, fn TxFn) error {
	return db.inner.WriteTX(ctx, fn)
}

func SQLTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05")
}
