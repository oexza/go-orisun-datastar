package todo

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

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

	model, err := loadRenameTodoContext(ctx, retriever, command.TodoID, command.UserRegisteredID)
	if err != nil {
		return RenameTodoResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return RenameTodoResult{}, err
	}
	if model.title == title {
		return RenameTodoResult{Skipped: true}, nil
	}

	eventID := uuidv7.NewString()
	event := NewTodoRenamedEvent(eventID, command.TodoID, command.UserRegisteredID, title, time.Now(), nil)

	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return RenameTodoResult{}, err
	}
	return RenameTodoResult{TodoRenamedID: eventID}, nil
}

type renameTodoContext struct {
	exists   bool
	deleted  bool
	title    string
	position eventstore.Position
	events   []eventstore.ResolvedEvent
	query    eventstore.Query
}

func loadRenameTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*renameTodoContext, error) {
	query := streamQuery(todoID, userRegisteredID)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)

	model := &renameTodoContext{position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *renameTodoContext) requireActive() error {
	if !m.exists || m.deleted {
		return eventstore.ErrNotFound
	}
	return nil
}

func (m *renameTodoContext) handle(resolved eventstore.ResolvedEvent) {
	data := resolved.Event.Data
	switch resolved.Event.EventType {
	case TodoCreated:
		m.exists = true
		m.deleted = false
		m.title, _ = data["title"].(string)
	case TodoRenamed:
		m.title, _ = data["title"].(string)
	case TodoDeleted:
		m.deleted = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
