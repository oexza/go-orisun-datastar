package todo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type CreateTodoCommand struct {
	UserRegisteredID string
	Title            string
	Metadata         CommandMetadata
}

type CreateTodoResult struct {
	TodoID string
}

func CreateTodoCommandHandler(ctx context.Context, command CreateTodoCommand, saver eventstore.Saver) (CreateTodoResult, error) {
	title, err := validateTitle(command.Title)
	if err != nil {
		return CreateTodoResult{}, err
	}

	todoID := uuid.NewString()
	query := streamQuery(todoID, command.UserRegisteredID)
	event := NewTodoCreatedEvent(todoID, command.UserRegisteredID, title, time.Now(), metadataWithQuery(command.Metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, query); err != nil {
		return CreateTodoResult{}, err
	}
	return CreateTodoResult{TodoID: todoID}, nil
}
