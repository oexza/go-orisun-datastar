package todo

import (
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const (
	TodoCreated   = "TodoCreated"
	TodoRenamed   = "TodoRenamed"
	TodoCompleted = "TodoCompleted"
	TodoReopened  = "TodoReopened"
	TodoDeleted   = "TodoDeleted"
)

const (
	TodoIDField                    = "todoId"
	TodoUserRegisteredIDField      = "userRegisteredId"
	TodoTitleField                 = "title"
	TodoCreatedAtField             = "createdAt"
	TodoRenamedIDField             = "todoRenamedId"
	TodoRenamedAtField             = "renamedAt"
	TodoCompletedIDField           = "todoCompletedId"
	TodoCompletedAtField           = "completedAt"
	TodoReopenedIDField            = "todoReopenedId"
	TodoReopenedAtField            = "reopenedAt"
	TodoDeletedIDField             = "todoDeletedId"
	TodoDeletedAtField             = "deletedAt"
	TodoScopeIDField               = "scope.todoId"
	TodoScopeUserRegisteredIDField = "scope.userRegisteredId"
)

type TodoCreatedEvent struct {
	TodoID           string    `json:"todoId"`
	UserRegisteredID string    `json:"userRegisteredId"`
	Title            string    `json:"title"`
	CreatedAt        string    `json:"createdAt"`
	Scope            TodoScope `json:"scope"`
}

type TodoRenamedEvent struct {
	TodoRenamedID string    `json:"todoRenamedId"`
	Title         string    `json:"title"`
	RenamedAt     string    `json:"renamedAt"`
	Scope         TodoScope `json:"scope"`
}

type TodoCompletedEvent struct {
	TodoCompletedID string    `json:"todoCompletedId"`
	CompletedAt     string    `json:"completedAt"`
	Scope           TodoScope `json:"scope"`
}

type TodoReopenedEvent struct {
	TodoReopenedID string    `json:"todoReopenedId"`
	ReopenedAt     string    `json:"reopenedAt"`
	Scope          TodoScope `json:"scope"`
}

type TodoDeletedEvent struct {
	TodoDeletedID string    `json:"todoDeletedId"`
	DeletedAt     string    `json:"deletedAt"`
	Scope         TodoScope `json:"scope"`
}

type TodoScope struct {
	TodoID           string `json:"todoId"`
	UserRegisteredID string `json:"userRegisteredId"`
}

func NewTodoCreatedEvent(todoID, userRegisteredID, title string, createdAt time.Time, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   todoID,
		EventType: TodoCreated,
		Data: eventstore.MustData(TodoCreatedEvent{
			TodoID:           todoID,
			UserRegisteredID: userRegisteredID,
			Title:            title,
			CreatedAt:        createdAt.Format(time.RFC3339),
			Scope:            todoScope(todoID, userRegisteredID),
		}),
		Metadata: metadata,
	}
}

func NewTodoRenamedEvent(todoRenamedID, todoID, userRegisteredID, title string, renamedAt time.Time, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   todoRenamedID,
		EventType: TodoRenamed,
		Data: eventstore.MustData(TodoRenamedEvent{
			TodoRenamedID: todoRenamedID,
			Title:         title,
			RenamedAt:     renamedAt.Format(time.RFC3339),
			Scope:         todoScope(todoID, userRegisteredID),
		}),
		Metadata: metadata,
	}
}

func NewTodoCompletedEvent(todoCompletedID, todoID, userRegisteredID string, completedAt time.Time, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   todoCompletedID,
		EventType: TodoCompleted,
		Data: eventstore.MustData(TodoCompletedEvent{
			TodoCompletedID: todoCompletedID,
			CompletedAt:     completedAt.Format(time.RFC3339),
			Scope:           todoScope(todoID, userRegisteredID),
		}),
		Metadata: metadata,
	}
}

func NewTodoReopenedEvent(todoReopenedID, todoID, userRegisteredID string, reopenedAt time.Time, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   todoReopenedID,
		EventType: TodoReopened,
		Data: eventstore.MustData(TodoReopenedEvent{
			TodoReopenedID: todoReopenedID,
			ReopenedAt:     reopenedAt.Format(time.RFC3339),
			Scope:          todoScope(todoID, userRegisteredID),
		}),
		Metadata: metadata,
	}
}

func NewTodoDeletedEvent(todoDeletedID, todoID, userRegisteredID string, deletedAt time.Time, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   todoDeletedID,
		EventType: TodoDeleted,
		Data: eventstore.MustData(TodoDeletedEvent{
			TodoDeletedID: todoDeletedID,
			DeletedAt:     deletedAt.Format(time.RFC3339),
			Scope:         todoScope(todoID, userRegisteredID),
		}),
		Metadata: metadata,
	}
}

func todoScope(todoID, userRegisteredID string) TodoScope {
	return TodoScope{TodoID: todoID, UserRegisteredID: userRegisteredID}
}
