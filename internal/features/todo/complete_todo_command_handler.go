package todo

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

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

	eventID := uuidv7.NewString()
	event := NewTodoCompletedEvent(eventID, command.TodoID, command.UserRegisteredID, time.Now(), nil)

	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
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
	query     eventstore.Query
}

func loadCompleteTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*completeTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)

	model := &completeTodoContext{position: latest.ContextPosition, events: events, query: query}
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
