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
	model, err := loadDeleteTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
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

type deleteTodoContext struct {
	exists   bool
	deleted  bool
	position eventstore.Position
	events   []eventstore.ResolvedEvent
}

func loadDeleteTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*deleteTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	events, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 100, eventstore.Forward, query)
	if err != nil {
		return nil, err
	}

	model := &deleteTodoContext{position: eventstore.NoEventPosition, events: events}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *deleteTodoContext) requireActive() error {
	if !m.exists || m.deleted {
		return eventstore.ErrNotFound
	}
	return nil
}

func (m *deleteTodoContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case TodoCreated:
		m.exists = true
		m.deleted = false
	case TodoDeleted:
		m.deleted = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
