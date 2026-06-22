package todo

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

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

	eventID := uuidv7.NewString()
	event := NewTodoDeletedEvent(eventID, command.TodoID, command.UserRegisteredID, time.Now(), nil)

	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return DeleteTodoResult{}, err
	}
	return DeleteTodoResult{TodoDeletedID: eventID}, nil
}

type deleteTodoContext struct {
	exists   bool
	deleted  bool
	position eventstore.Position
	events   []eventstore.ResolvedEvent
	query    eventstore.Query
}

func loadDeleteTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*deleteTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)

	model := &deleteTodoContext{position: latest.ContextPosition, events: events, query: query}
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
