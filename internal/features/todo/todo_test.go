package todo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

func TestRenameTodoCommandSkipsUnchangedTitle(t *testing.T) {
	t.Parallel()

	store := newFakeTodoStore("user-1",
		NewTodoCreatedEvent("todo-1", "user-1", "Ship starter", time.Now(), nil),
	)

	result, err := RenameTodoCommandHandler(context.Background(), RenameTodoCommand{
		UserRegisteredID: "user-1",
		TodoID:           "todo-1",
		Title:            "  Ship starter  ",
	}, store, store)
	if err != nil {
		t.Fatalf("rename todo: %v", err)
	}
	if !result.Skipped {
		t.Fatal("expected unchanged rename to be skipped")
	}
	if got := store.countSaved(TodoRenamed); got != 0 {
		t.Fatalf("expected no rename event, got %d", got)
	}
}

func TestBulkTodoActionsAppendExpectedEvents(t *testing.T) {
	t.Parallel()

	readModel := fakeTodoReadModel{todos: []views.Todo{
		{TodoID: "active-1", Title: "Active one"},
		{TodoID: "active-2", Title: "Active two"},
		{TodoID: "done-1", Title: "Done one", Completed: true},
	}}
	store := newFakeTodoStore("user-1",
		NewTodoCreatedEvent("active-1", "user-1", "Active one", time.Now(), nil),
		NewTodoCreatedEvent("active-2", "user-1", "Active two", time.Now(), nil),
		NewTodoCreatedEvent("done-1", "user-1", "Done one", time.Now(), nil),
		NewTodoCompletedEvent("done-1-completed", "done-1", "user-1", time.Now(), nil),
	)

	if err := CompleteAllActiveTodosCommandHandler(context.Background(), CompleteAllActiveTodosCommand{
		UserRegisteredID: "user-1",
	}, readModel, store, store); err != nil {
		t.Fatalf("complete active todos: %v", err)
	}
	if got := store.countSaved(TodoCompleted); got != 2 {
		t.Fatalf("expected two completed events, got %d", got)
	}

	if err := ClearCompletedTodosCommandHandler(context.Background(), ClearCompletedTodosCommand{
		UserRegisteredID: "user-1",
	}, readModel, store, store); err != nil {
		t.Fatalf("clear completed todos: %v", err)
	}
	if got := store.countSaved(TodoDeleted); got != 1 {
		t.Fatalf("expected one deleted event, got %d", got)
	}
}

type fakeTodoReadModel struct {
	todos []views.Todo
	err   error
}

func (m fakeTodoReadModel) List(context.Context, string) ([]views.Todo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return append([]views.Todo(nil), m.todos...), nil
}

type fakeTodoStore struct {
	eventsByTodoID map[string][]eventstore.ResolvedEvent
	saved          []eventstore.DomainEvent
	position       int64
}

func newFakeTodoStore(userRegisteredID string, events ...eventstore.DomainEvent) *fakeTodoStore {
	store := &fakeTodoStore{eventsByTodoID: map[string][]eventstore.ResolvedEvent{}}
	for _, event := range events {
		store.appendResolved(userRegisteredID, event)
	}
	return store
}

func (s *fakeTodoStore) SaveEvents(_ context.Context, events []eventstore.DomainEvent, _ eventstore.Position, _ []eventstore.ResolvedEvent, _ eventstore.Query) (eventstore.WriteResult, error) {
	for _, event := range events {
		scope := eventstore.Scope(event.Data)
		userRegisteredID, _ := scope["userRegisteredId"].(string)
		s.appendResolved(userRegisteredID, event)
		s.saved = append(s.saved, event)
	}
	return eventstore.WriteResult{Position: eventstore.Position{Commit: s.position, Prepare: s.position}}, nil
}

func (s *fakeTodoStore) GetEvents(_ context.Context, _ eventstore.Position, _ int, _ eventstore.Direction, query eventstore.Query) ([]eventstore.ResolvedEvent, error) {
	todoID, ok := todoIDFromQuery(query)
	if !ok {
		return nil, errors.New("missing todo scope query")
	}
	return append([]eventstore.ResolvedEvent(nil), s.eventsByTodoID[todoID]...), nil
}

func (s *fakeTodoStore) GetLatestByCriteria(_ context.Context, criteria []eventstore.Criterion) (eventstore.LatestByCriteriaResult, error) {
	query := eventstore.Query{Criteria: criteria}
	todoID, ok := todoIDFromQuery(query)
	if !ok {
		return eventstore.LatestByCriteriaResult{}, errors.New("missing todo scope query")
	}
	result := eventstore.LatestByCriteriaResult{
		Results:         make([]eventstore.LatestCriterionResult, 0, len(criteria)),
		ContextPosition: eventstore.NoEventPosition,
	}
	for _, criterion := range criteria {
		latest := eventstore.LatestCriterionResult{Criterion: criterion}
		for _, event := range s.eventsByTodoID[todoID] {
			if !matchesCriterion(event, criterion) {
				continue
			}
			candidate := event
			if latest.Event == nil || candidate.Position.After(latest.Event.Position) {
				latest.Event = &candidate
			}
		}
		if latest.Event != nil && latest.Event.Position.After(result.ContextPosition) {
			result.ContextPosition = latest.Event.Position
		}
		result.Results = append(result.Results, latest)
	}
	return result, nil
}

func (s *fakeTodoStore) appendResolved(fallbackUserRegisteredID string, event eventstore.DomainEvent) {
	scope := eventstore.Scope(event.Data)
	todoID, _ := scope["todoId"].(string)
	if todoID == "" {
		todoID, _ = event.Data["todoId"].(string)
	}
	if scope["userRegisteredId"] == nil && fallbackUserRegisteredID != "" {
		scope["userRegisteredId"] = fallbackUserRegisteredID
		event.Data["scope"] = scope
	}
	s.position++
	resolved := eventstore.ResolvedEvent{
		Position: eventstore.Position{Commit: s.position, Prepare: s.position},
		Event:    event,
	}
	s.eventsByTodoID[todoID] = append(s.eventsByTodoID[todoID], resolved)
}

func matchesCriterion(resolved eventstore.ResolvedEvent, criterion eventstore.Criterion) bool {
	for _, tag := range criterion.Tags {
		if tag.Key == "eventType" {
			if resolved.Event.EventType != tag.Value {
				return false
			}
			continue
		}
		value := stringDataValue(resolved.Event.Data, tag.Key)
		if value != tag.Value {
			return false
		}
	}
	return true
}

func stringDataValue(data map[string]any, key string) string {
	current := any(data)
	for _, part := range strings.Split(key, ".") {
		mapped, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = mapped[part]
	}
	value, _ := current.(string)
	return value
}

func (s *fakeTodoStore) countSaved(eventType string) int {
	var count int
	for _, event := range s.saved {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func todoIDFromQuery(query eventstore.Query) (string, bool) {
	for _, criterion := range query.Criteria {
		for _, tag := range criterion.Tags {
			if tag.Key == TodoScopeIDField {
				return tag.Value, true
			}
		}
	}
	return "", false
}
