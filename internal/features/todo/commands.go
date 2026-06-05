package todo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/go-orisun-datastar/internal/eventstore"
)

type CommandMetadata = eventstore.CommandMetadata

type CreateTodoCommand struct {
	UserRegisteredID string
	Title            string
	Metadata         CommandMetadata
}

type CreateTodoResult struct {
	TodoID string
}

func CreateTodoCommandHandler(ctx context.Context, command CreateTodoCommand, saver eventstore.Saver) (CreateTodoResult, error) {
	title, err := validateTitle(command.Title)
	if err != nil {
		return CreateTodoResult{}, err
	}

	todoID := uuid.NewString()
	query := streamQuery(todoID, command.UserRegisteredID)
	event := NewTodoCreatedEvent(todoID, command.UserRegisteredID, title, time.Now(), metadataWithQuery(command.Metadata, query))

	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, query); err != nil {
		return CreateTodoResult{}, err
	}
	return CreateTodoResult{TodoID: todoID}, nil
}

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
	result, err := changeTodoCompletion(ctx, changeTodoCompletionCommand{
		userRegisteredID: command.UserRegisteredID,
		todoID:           command.TodoID,
		complete:         true,
		metadata:         command.Metadata,
	}, saver, retriever)
	if err != nil {
		return CompleteTodoResult{}, err
	}
	return CompleteTodoResult{TodoCompletedID: result.eventID, Skipped: result.skipped}, nil
}

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
	result, err := changeTodoCompletion(ctx, changeTodoCompletionCommand{
		userRegisteredID: command.UserRegisteredID,
		todoID:           command.TodoID,
		complete:         false,
		metadata:         command.Metadata,
	}, saver, retriever)
	if err != nil {
		return ReopenTodoResult{}, err
	}
	return ReopenTodoResult{TodoReopenedID: result.eventID, Skipped: result.skipped}, nil
}

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
