package todo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type RenameTodoCommand struct {
	UserRegisteredID string
	TodoID           string
	Title            string
	Metadata         CommandMetadata
}

type RenameTodoResult struct {
	TodoRenamedID string
	Skipped       bool
}

func RenameTodoCommandHandler(ctx context.Context, command RenameTodoCommand, saver eventstore.Saver, retriever eventstore.Retriever) (RenameTodoResult, error) {
	title, err := validateTitle(command.Title)
	if err != nil {
		return RenameTodoResult{}, err
	}

	model, err := loadTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
	if err != nil {
		return RenameTodoResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return RenameTodoResult{}, err
	}
	if model.title == title {
		return RenameTodoResult{Skipped: true}, nil
	}

	query := streamQuery(command.TodoID, command.UserRegisteredID)
	eventID := uuid.NewString()
	event := NewTodoRenamedEvent(eventID, command.TodoID, command.UserRegisteredID, title, time.Now(), metadataWithQuery(command.Metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, query); err != nil {
		return RenameTodoResult{}, err
	}
	return RenameTodoResult{TodoRenamedID: eventID}, nil
}
