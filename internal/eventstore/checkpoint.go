package eventstore

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCheckpointer struct {
	db *pgxpool.Pool
}

func NewPostgresCheckpointer(db *pgxpool.Pool) *PostgresCheckpointer {
	return &PostgresCheckpointer{db: db}
}

func (c *PostgresCheckpointer) GetCheckpoint(ctx context.Context, name string) (Position, bool, error) {
	var position Position
	err := c.db.QueryRow(ctx, `
		SELECT commit_position::bigint, prepare_position::bigint
		FROM projector_checkpoint
		WHERE name = $1
	`, name).Scan(&position.Commit, &position.Prepare)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return NoEventPosition, false, nil
		}
		return NoEventPosition, false, err
	}
	return position, true, nil
}

func (c *PostgresCheckpointer) UpdateCheckpoint(ctx context.Context, name string, position Position) error {
	_, err := c.db.Exec(ctx, `
		INSERT INTO projector_checkpoint (id, name, commit_position, prepare_position, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (name) DO UPDATE
		SET commit_position = EXCLUDED.commit_position,
		    prepare_position = EXCLUDED.prepare_position,
		    updated_at = now()
	`, uuid.NewString(), name, position.Commit, position.Prepare)
	return err
}
