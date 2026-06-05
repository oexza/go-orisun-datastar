package eventstore

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"zombiezen.com/go/sqlite"
)

type SQLiteCheckpointer struct {
	db *appdb.DB
}

func NewSQLiteCheckpointer(db *appdb.DB) *SQLiteCheckpointer {
	return &SQLiteCheckpointer{db: db}
}

func (c *SQLiteCheckpointer) GetCheckpoint(ctx context.Context, name string) (Position, bool, error) {
	var row *dbsql.GetEventHandlerCheckpointRes
	if err := c.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceGetEventHandlerCheckpoint(conn, name)
		return err
	}); err != nil {
		return NoEventPosition, false, err
	}
	if row == nil {
		return NoEventPosition, false, nil
	}
	return Position{Commit: row.CommitPosition, Prepare: row.PreparePosition}, true, nil
}

func (c *SQLiteCheckpointer) UpdateCheckpoint(ctx context.Context, name string, position Position) error {
	return c.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpsertEventHandlerCheckpoint(conn, dbsql.UpsertEventHandlerCheckpointParams{
			Id:              uuidv7.NewString(),
			Name:            name,
			CommitPosition:  position.Commit,
			PreparePosition: position.Prepare,
		})
	})
}
