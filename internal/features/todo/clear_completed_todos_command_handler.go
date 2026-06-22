package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type ClearCompletedTodosCommand struct {
	UserRegisteredID string
	Metadata         CommandMetadata
}

func ClearCompletedTodosCommandHandler(ctx context.Context, command ClearCompletedTodosCommand, readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	model, err := loadClearCompletedTodosContext(ctx, readModel, command.UserRegisteredID)
	if err != nil {
		return err
	}
	for _, item := range model.todos {
		if !item.Completed {
			continue
		}
		if _, err := DeleteTodoCommandHandler(ctx, DeleteTodoCommand{
			UserRegisteredID: command.UserRegisteredID,
			TodoID:           item.TodoID,
			Metadata:         command.Metadata,
		}, saver, retriever); err != nil {
			return err
		}
	}
	return nil
}

type clearCompletedTodosContext struct {
	todos []views.Todo
}

func loadClearCompletedTodosContext(ctx context.Context, readModel TodoReadModelReader, userRegisteredID string) (*clearCompletedTodosContext, error) {
	todos, err := readModel.List(ctx, userRegisteredID)
	if err != nil {
		return nil, err
	}
	return &clearCompletedTodosContext{todos: todos}, nil
}
