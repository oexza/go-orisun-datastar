package todo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type DeleteTodoCommand struct {
	UserRegisteredID string
	TodoID           string
	Metadata         CommandMetadata
}

type DeleteTodoResult struct {
	TodoDeletedID string
}

func DeleteTodoCommandHandler(ctx context.Context, command DeleteTodoCommand, saver eventstore.Saver, retriever eventstore.Retriever) (DeleteTodoResult, error) {
	model, err := loadTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
	if err != nil {
		return DeleteTodoResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return DeleteTodoResult{}, err
	}

	query := streamQuery(command.TodoID, command.UserRegisteredID)
	eventID := uuid.NewString()
	event := NewTodoDeletedEvent(eventID, command.TodoID, command.UserRegisteredID, time.Now(), metadataWithQuery(command.Metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, query); err != nil {
		return DeleteTodoResult{}, err
	}
	return DeleteTodoResult{TodoDeletedID: eventID}, nil
}
