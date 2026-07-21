package todo

import (
	"context"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
)

type ReopenAllCompletedTodosCommand struct {
	UserRegisteredID string
	Metadata         CommandMetadata
}

func ReopenAllCompletedTodosCommandHandler(ctx context.Context, command ReopenAllCompletedTodosCommand, readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	model, err := loadReopenAllCompletedTodosContext(ctx, readModel, command.UserRegisteredID)
	if err != nil {
		return err
	}
	for _, item := range model.todos {
		if !item.Completed {
			continue
		}
		if _, err := ReopenTodoCommandHandler(ctx, ReopenTodoCommand{
			UserRegisteredID: command.UserRegisteredID,
			TodoID:           item.TodoID,
			Metadata:         command.Metadata,
		}, saver, retriever); err != nil {
			return err
		}
	}
	return nil
}

type reopenAllCompletedTodosContext struct {
	todos []views.Todo
}

func loadReopenAllCompletedTodosContext(ctx context.Context, readModel TodoReadModelReader, userRegisteredID string) (*reopenAllCompletedTodosContext, error) {
	todos, err := readModel.List(ctx, userRegisteredID)
	if err != nil {
		return nil, err
	}
	return &reopenAllCompletedTodosContext{todos: todos}, nil
}
