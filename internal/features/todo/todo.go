package todo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/views"
)

const (
	TodoCreated   = "TodoCreated"
	TodoRenamed   = "TodoRenamed"
	TodoCompleted = "TodoCompleted"
	TodoReopened  = "TodoReopened"
	TodoDeleted   = "TodoDeleted"
)

type Service struct {
	db        *pgxpool.Pool
	saver     eventstore.Saver
	retriever eventstore.Retriever
	publisher eventstore.Publisher
}

func NewService(db *pgxpool.Pool, saver eventstore.Saver, retriever eventstore.Retriever, publisher eventstore.Publisher) *Service {
	return &Service{db: db, saver: saver, retriever: retriever, publisher: publisher}
}

func Channel(userRegisteredID string) string {
	return "todo." + userRegisteredID
}

func (s *Service) List(ctx context.Context, userRegisteredID string) ([]views.Todo, error) {
	rows, err := s.db.Query(ctx, `
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

func (s *Service) Create(ctx context.Context, userRegisteredID, title string) (string, error) {
	result, err := CreateTodoCommandHandler(ctx, CreateTodoCommand{
		UserRegisteredID: userRegisteredID,
		Title:            title,
	}, s.saver)
	return result.TodoID, err
}

func (s *Service) Rename(ctx context.Context, userRegisteredID, todoID, title string) error {
	_, err := RenameTodoCommandHandler(ctx, RenameTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
		Title:            title,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Complete(ctx context.Context, userRegisteredID, todoID string) error {
	_, err := CompleteTodoCommandHandler(ctx, CompleteTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Reopen(ctx context.Context, userRegisteredID, todoID string) error {
	_, err := ReopenTodoCommandHandler(ctx, ReopenTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Delete(ctx context.Context, userRegisteredID, todoID string) error {
	_, err := DeleteTodoCommandHandler(ctx, DeleteTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
	}, s.saver, s.retriever)
	return err
}
