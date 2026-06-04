package eventstore

import (
	"context"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/google/uuid"
)

type SQLiteCheckpointer struct {
	db *appdb.DB
}

func NewSQLiteCheckpointer(db *appdb.DB) *SQLiteCheckpointer {
	return &SQLiteCheckpointer{db: db}
}

func (c *SQLiteCheckpointer) GetCheckpoint(ctx context.Context, name string) (Position, bool, error) {
	var position Position
	err := c.db.QueryRow(ctx, `
		SELECT commit_position, prepare_position
		FROM projector_checkpoint
		WHERE name = $1
	`, name).Scan(&position.Commit, &position.Prepare)
	if err != nil {
		if err == appdb.ErrNoRows {
			return NoEventPosition, false, nil
		}
		return NoEventPosition, false, err
	}
	return position, true, nil
}

func (c *SQLiteCheckpointer) UpdateCheckpoint(ctx context.Context, name string, position Position) error {
	_, err := c.db.Exec(ctx, `
		INSERT INTO projector_checkpoint (id, name, commit_position, prepare_position, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (name) DO UPDATE
		SET commit_position = EXCLUDED.commit_position,
		    prepare_position = EXCLUDED.prepare_position,
		    updated_at = CURRENT_TIMESTAMP
	`, uuid.NewString(), name, position.Commit, position.Prepare)
	return err
}
