package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	migrations := os.DirFS("migrations")
	entries, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)
	for _, entry := range entries {
		sql, err := fs.ReadFile(migrations, entry)
		if err != nil {
			return err
		}
		if _, err := db.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("run migration %s: %w", entry, err)
		}
	}
	return nil
}
