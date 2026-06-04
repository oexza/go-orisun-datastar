package todo

import (
	"context"
	"time"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/example/hono-event-starter-go/internal/views"
)

type ReadModel struct {
	db *appdb.DB
}

func NewReadModel(db *appdb.DB) *ReadModel {
	return &ReadModel{db: db}
}

func (m *ReadModel) List(ctx context.Context, userRegisteredID string) ([]views.Todo, error) {
	rows, err := m.db.Query(ctx, `
		SELECT todo_id, title, completed, created_at, updated_at
		FROM todo_items
		WHERE user_registered_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC, todo_id DESC
	`, userRegisteredID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []views.Todo
	for rows.Next() {
		var todo views.Todo
		if err := rows.Scan(&todo.TodoID, &todo.Title, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

func (m *ReadModel) InsertCreatedTodo(ctx context.Context, event TodoCreatedProjection) error {
	_, err := m.db.Exec(ctx, `
		INSERT INTO todo_items (todo_id, user_registered_id, title, completed, completed_at, deleted_at, last_event_commit_position, last_event_prepare_position, created_at, updated_at)
		VALUES ($1, $2, $3, false, null, null, $4, $5, $6, $6)
		ON CONFLICT (todo_id) DO NOTHING
	`, event.TodoID, event.UserRegisteredID, event.Title, event.Position.Commit, event.Position.Prepare, event.CreatedAt)
	return err
}

func (m *ReadModel) RenameTodo(ctx context.Context, event TodoRenamedProjection) error {
	_, err := m.db.Exec(ctx, `
		UPDATE todo_items
		SET title = $1,
		    last_event_commit_position = $2,
		    last_event_prepare_position = $3,
		    updated_at = $4
		WHERE todo_id = $5
	`, event.Title, event.Position.Commit, event.Position.Prepare, event.RenamedAt, event.TodoID)
	return err
}

func (m *ReadModel) CompleteTodo(ctx context.Context, event TodoCompletedProjection) error {
	_, err := m.db.Exec(ctx, `
		UPDATE todo_items
		SET completed = true,
		    completed_at = $1,
		    last_event_commit_position = $2,
		    last_event_prepare_position = $3,
		    updated_at = $1
		WHERE todo_id = $4
	`, event.CompletedAt, event.Position.Commit, event.Position.Prepare, event.TodoID)
	return err
}

func (m *ReadModel) ReopenTodo(ctx context.Context, event TodoReopenedProjection) error {
	_, err := m.db.Exec(ctx, `
		UPDATE todo_items
		SET completed = false,
		    completed_at = null,
		    last_event_commit_position = $1,
		    last_event_prepare_position = $2,
		    updated_at = $3
		WHERE todo_id = $4
	`, event.Position.Commit, event.Position.Prepare, event.ReopenedAt, event.TodoID)
	return err
}

func (m *ReadModel) DeleteTodo(ctx context.Context, event TodoDeletedProjection) error {
	_, err := m.db.Exec(ctx, `
		UPDATE todo_items
		SET deleted_at = $1,
		    last_event_commit_position = $2,
		    last_event_prepare_position = $3,
		    updated_at = $1
		WHERE todo_id = $4
	`, event.DeletedAt, event.Position.Commit, event.Position.Prepare, event.TodoID)
	return err
}

func parseTime(value any) time.Time {
	text, _ := value.(string)
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Now()
	}
	return parsed
}
