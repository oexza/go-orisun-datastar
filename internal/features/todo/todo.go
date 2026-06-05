package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
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

func (s *Service) Create(ctx context.Context, userRegisteredID, title string, metadata ...CommandMetadata) (string, error) {
	return s.CreateWithMetadata(ctx, userRegisteredID, title, firstMetadata(metadata))
}

func (s *Service) CreateWithMetadata(ctx context.Context, userRegisteredID, title string, metadata CommandMetadata) (string, error) {
	result, err := CreateTodoCommandHandler(ctx, CreateTodoCommand{
		UserRegisteredID: userRegisteredID,
		Title:            title,
		Metadata:         metadata,
	}, s.saver)
	return result.TodoID, err
}

func (s *Service) Rename(ctx context.Context, userRegisteredID, todoID, title string, metadata ...CommandMetadata) error {
	return s.RenameWithMetadata(ctx, userRegisteredID, todoID, title, firstMetadata(metadata))
}

func (s *Service) RenameWithMetadata(ctx context.Context, userRegisteredID, todoID, title string, metadata CommandMetadata) error {
	_, err := RenameTodoCommandHandler(ctx, RenameTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
		Title:            title,
		Metadata:         metadata,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Complete(ctx context.Context, userRegisteredID, todoID string, metadata ...CommandMetadata) error {
	return s.CompleteWithMetadata(ctx, userRegisteredID, todoID, firstMetadata(metadata))
}

func (s *Service) CompleteWithMetadata(ctx context.Context, userRegisteredID, todoID string, metadata CommandMetadata) error {
	_, err := CompleteTodoCommandHandler(ctx, CompleteTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
		Metadata:         metadata,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Reopen(ctx context.Context, userRegisteredID, todoID string, metadata ...CommandMetadata) error {
	return s.ReopenWithMetadata(ctx, userRegisteredID, todoID, firstMetadata(metadata))
}

func (s *Service) ReopenWithMetadata(ctx context.Context, userRegisteredID, todoID string, metadata CommandMetadata) error {
	_, err := ReopenTodoCommandHandler(ctx, ReopenTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
		Metadata:         metadata,
	}, s.saver, s.retriever)
	return err
}

func (s *Service) Delete(ctx context.Context, userRegisteredID, todoID string, metadata ...CommandMetadata) error {
	return s.DeleteWithMetadata(ctx, userRegisteredID, todoID, firstMetadata(metadata))
}

func (s *Service) DeleteWithMetadata(ctx context.Context, userRegisteredID, todoID string, metadata CommandMetadata) error {
	_, err := DeleteTodoCommandHandler(ctx, DeleteTodoCommand{
		UserRegisteredID: userRegisteredID,
		TodoID:           todoID,
		Metadata:         metadata,
	}, s.saver, s.retriever)
	return err
}

func firstMetadata(metadata []CommandMetadata) CommandMetadata {
	if len(metadata) == 0 || metadata[0] == nil {
		return CommandMetadata{}
	}
	return metadata[0]
}
