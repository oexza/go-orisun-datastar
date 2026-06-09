package todo

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

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
	model, err := newCreateTodoContext(command)
	if err != nil {
		return CreateTodoResult{}, err
	}

	event := NewTodoCreatedEvent(model.todoID, command.UserRegisteredID, model.title, time.Now(), metadataWithQuery(command.Metadata, model.query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, model.query); err != nil {
		return CreateTodoResult{}, err
	}
	return CreateTodoResult{TodoID: model.todoID}, nil
}

type createTodoContext struct {
	todoID string
	title  string
	query  eventstore.Query
}

func newCreateTodoContext(command CreateTodoCommand) (*createTodoContext, error) {
	title, err := validateTitle(command.Title)
	if err != nil {
		return nil, err
	}
	todoID := uuidv7.NewString()
	return &createTodoContext{
		todoID: todoID,
		title:  title,
		query:  streamQuery(todoID, command.UserRegisteredID),
	}, nil
}
