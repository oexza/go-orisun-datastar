package todo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type CompleteTodoCommand struct {
	UserRegisteredID string
	TodoID           string
	Metadata         CommandMetadata
}

type CompleteTodoResult struct {
	TodoCompletedID string
	Skipped         bool
}

func CompleteTodoCommandHandler(ctx context.Context, command CompleteTodoCommand, saver eventstore.Saver, retriever eventstore.Retriever) (CompleteTodoResult, error) {
	model, err := loadCompleteTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
	if err != nil {
		return CompleteTodoResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return CompleteTodoResult{}, err
	}
	if model.completed {
		return CompleteTodoResult{Skipped: true}, nil
	}

	query := streamQuery(command.TodoID, command.UserRegisteredID)
	eventID := uuid.NewString()
	event := NewTodoCompletedEvent(eventID, command.TodoID, command.UserRegisteredID, time.Now(), metadataWithQuery(command.Metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, query); err != nil {
		return CompleteTodoResult{}, err
	}
	return CompleteTodoResult{TodoCompletedID: eventID}, nil
}

type completeTodoContext struct {
	exists    bool
	deleted   bool
	completed bool
	position  eventstore.Position
	events    []eventstore.ResolvedEvent
}

func loadCompleteTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*completeTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	events, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 100, eventstore.Forward, query)
	if err != nil {
		return nil, err
	}

	model := &completeTodoContext{position: eventstore.NoEventPosition, events: events}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *completeTodoContext) requireActive() error {
	if !m.exists || m.deleted {
		return eventstore.ErrNotFound
	}
	return nil
}

func (m *completeTodoContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case TodoCreated:
		m.exists = true
		m.deleted = false
		m.completed = false
	case TodoCompleted:
		m.completed = true
	case TodoReopened:
		m.completed = false
	case TodoDeleted:
		m.deleted = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
