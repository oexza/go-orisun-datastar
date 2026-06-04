package todo

import (
	"context"

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
	readModel TodoReadModelReader
	saver     eventstore.Saver
	retriever eventstore.Retriever
	publisher eventstore.Publisher
}

func NewService(readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever, publisher eventstore.Publisher) *Service {
	return &Service{readModel: readModel, saver: saver, retriever: retriever, publisher: publisher}
}

func Channel(userRegisteredID string) string {
	return "todo." + userRegisteredID
}

func (s *Service) List(ctx context.Context, userRegisteredID string) ([]views.Todo, error) {
	return s.readModel.List(ctx, userRegisteredID)
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
