package todo

import (
	"context"
	"time"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/example/hono-event-starter-go/internal/dbsql"
	"github.com/example/hono-event-starter-go/internal/views"
	"zombiezen.com/go/sqlite"
)

type ReadModel struct {
	db *appdb.DB
}

func NewReadModel(db *appdb.DB) *ReadModel {
	return &ReadModel{db: db}
}

func (m *ReadModel) List(ctx context.Context, userRegisteredID string) ([]views.Todo, error) {
	var rows []dbsql.ListTodosRes
	if err := m.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		rows, err = dbsql.OnceListTodos(conn, userRegisteredID)
		return err
	}); err != nil {
		return nil, err
	}
	todos := make([]views.Todo, 0, len(rows))
	for _, row := range rows {
		todos = append(todos, views.Todo{
			TodoID:    row.TodoId,
			Title:     row.Title,
			Completed: row.Completed != 0,
			CreatedAt: parseDBTime(row.CreatedAt),
			UpdatedAt: parseDBTime(row.UpdatedAt),
		})
	}
	return todos, nil
}

func (m *ReadModel) InsertCreatedTodo(ctx context.Context, event TodoCreatedProjection) error {
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceInsertCreatedTodo(conn, dbsql.InsertCreatedTodoParams{
			TodoId:                   event.TodoID,
			UserRegisteredId:         event.UserRegisteredID,
			Title:                    event.Title,
			LastEventCommitPosition:  event.Position.Commit,
			LastEventPreparePosition: event.Position.Prepare,
			CreatedAt:                appdb.SQLTime(event.CreatedAt),
		})
	})
}

func (m *ReadModel) RenameTodo(ctx context.Context, event TodoRenamedProjection) error {
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceRenameTodo(conn, dbsql.RenameTodoParams{
			Title:                    event.Title,
			LastEventCommitPosition:  event.Position.Commit,
			LastEventPreparePosition: event.Position.Prepare,
			UpdatedAt:                appdb.SQLTime(event.RenamedAt),
			TodoId:                   event.TodoID,
		})
	})
}

func (m *ReadModel) CompleteTodo(ctx context.Context, event TodoCompletedProjection) error {
	completedAt := appdb.SQLTime(event.CompletedAt)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCompleteTodo(conn, dbsql.CompleteTodoParams{
			CompletedAt:              &completedAt,
			LastEventCommitPosition:  event.Position.Commit,
			LastEventPreparePosition: event.Position.Prepare,
			TodoId:                   event.TodoID,
		})
	})
}

func (m *ReadModel) ReopenTodo(ctx context.Context, event TodoReopenedProjection) error {
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceReopenTodo(conn, dbsql.ReopenTodoParams{
			LastEventCommitPosition:  event.Position.Commit,
			LastEventPreparePosition: event.Position.Prepare,
			UpdatedAt:                appdb.SQLTime(event.ReopenedAt),
			TodoId:                   event.TodoID,
		})
	})
}

func (m *ReadModel) DeleteTodo(ctx context.Context, event TodoDeletedProjection) error {
	deletedAt := appdb.SQLTime(event.DeletedAt)
	return m.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceDeleteTodo(conn, dbsql.DeleteTodoParams{
			DeletedAt:                &deletedAt,
			LastEventCommitPosition:  event.Position.Commit,
			LastEventPreparePosition: event.Position.Prepare,
			TodoId:                   event.TodoID,
		})
	})
}

func parseTime(value any) time.Time {
	text, _ := value.(string)
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Now()
	}
	return parsed
}

func parseDBTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}
