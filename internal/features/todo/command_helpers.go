package todo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type changeTodoCompletionCommand struct {
	userRegisteredID string
	todoID           string
	complete         bool
	metadata         CommandMetadata
}

type changeTodoCompletionResult struct {
	eventID string
	skipped bool
}

func changeTodoCompletion(ctx context.Context, command changeTodoCompletionCommand, saver eventstore.Saver, retriever eventstore.Retriever) (changeTodoCompletionResult, error) {
	model, err := loadTodoContext(ctx, retriever, command.todoID, command.userRegisteredID)
	if err != nil {
		return changeTodoCompletionResult{}, err
	}
	if err := model.requireActive(); err != nil {
		return changeTodoCompletionResult{}, err
	}
	if model.completed == command.complete {
		return changeTodoCompletionResult{skipped: true}, nil
	}

	newEvent := NewTodoReopenedEvent
	if command.complete {
		newEvent = NewTodoCompletedEvent
	}

	query := streamQuery(command.todoID, command.userRegisteredID)
	eventID := uuid.NewString()
	event := newEvent(eventID, command.todoID, command.userRegisteredID, time.Now(), metadataWithQuery(command.metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, query); err != nil {
		return changeTodoCompletionResult{}, err
	}
	return changeTodoCompletionResult{eventID: eventID}, nil
}

type todoContextModel struct {
	exists    bool
	deleted   bool
	completed bool
	title     string
	position  eventstore.Position
	events    []eventstore.ResolvedEvent
}

func loadTodoContext(ctx context.Context, retriever eventstore.Retriever, todoID, userRegisteredID string) (*todoContextModel, error) {
	events, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 100, eventstore.Forward, streamQuery(todoID, userRegisteredID))
	if err != nil {
		return nil, err
	}

	model := &todoContextModel{position: eventstore.NoEventPosition, events: events}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *todoContextModel) requireActive() error {
	if !m.exists || m.deleted {
		return eventstore.ErrNotFound
	}
	return nil
}

func (m *todoContextModel) handle(resolved eventstore.ResolvedEvent) {
	data := resolved.Event.Data
	switch resolved.Event.EventType {
	case TodoCreated:
		m.exists = true
		m.deleted = false
		m.completed = false
		m.title, _ = data["title"].(string)
	case TodoRenamed:
		m.title, _ = data["title"].(string)
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

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 160 {
		return "", errors.New("todo title must be between 1 and 160 characters")
	}
	return title, nil
}

func streamQuery(todoID, userRegisteredID string) eventstore.Query {
	criteria := make([]eventstore.Criterion, 0, 5)
	for _, eventType := range []string{TodoCreated, TodoRenamed, TodoCompleted, TodoReopened, TodoDeleted} {
		criteria = append(criteria, eventstore.Criterion{Tags: []eventstore.Tag{
			{Key: "eventType", Value: eventType},
			{Key: TodoScopeIDField, Value: todoID},
			{Key: TodoScopeUserRegisteredIDField, Value: userRegisteredID},
		}})
	}
	return eventstore.Query{Criteria: criteria}
}

func metadataWithQuery(metadata CommandMetadata, query eventstore.Query) map[string]any {
	return eventstore.MergeMetadata(map[string]any{"query": eventstore.MustJSON(query)}, metadata)
}
