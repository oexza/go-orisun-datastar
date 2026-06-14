package todo

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type ReopenTodoCommand struct {
	UserRegisteredID string
	TodoID           string
	Metadata         CommandMetadata
}

type ReopenTodoResult struct {
	TodoReopenedID string
	Skipped        bool
}

func ReopenTodoCommandHandler(ctx context.Context, command ReopenTodoCommand, saver eventstore.Saver, retriever eventstore.Retriever) (ReopenTodoResult, error) {
	model, err := loadReopenTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
	if err != nil {
		return ReopenTodoResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return ReopenTodoResult{}, err
	}
	if !model.completed {
		return ReopenTodoResult{Skipped: true}, nil
	}

	eventID := uuidv7.NewString()
	event := NewTodoReopenedEvent(eventID, command.TodoID, command.UserRegisteredID, time.Now(), metadataWithQuery(command.Metadata, model.query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return ReopenTodoResult{}, err
	}
	return ReopenTodoResult{TodoReopenedID: eventID}, nil
}

type reopenTodoContext struct {
	exists    bool
	deleted   bool
	completed bool
	position  eventstore.Position
	events    []eventstore.ResolvedEvent
	query     eventstore.Query
}

func loadReopenTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*reopenTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)

	model := &reopenTodoContext{position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *reopenTodoContext) requireActive() error {
	if !m.exists || m.deleted {
		return eventstore.ErrNotFound
	}
	return nil
}

func (m *reopenTodoContext) handle(resolved eventstore.ResolvedEvent) {
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
