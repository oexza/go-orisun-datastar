package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/commandlimits"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type CompleteAllActiveTodosCommand struct {
	UserRegisteredID string
	Metadata         CommandMetadata
}

func CompleteAllActiveTodosCommandHandler(ctx context.Context, command CompleteAllActiveTodosCommand, readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	model, err := loadCompleteAllActiveTodosContext(ctx, readModel, command.UserRegisteredID)
	if err != nil {
		return err
	}
	for _, item := range model.todos {
		if item.Completed {
			continue
		}
		if _, err := CompleteTodoCommandHandler(ctx, CompleteTodoCommand{
			UserRegisteredID: command.UserRegisteredID,
			TodoID:           item.TodoID,
			Metadata:         command.Metadata,
		}, saver, retriever); err != nil {
			return err
		}
	}
	return nil
}

type completeAllActiveTodosContext struct {
	todos []views.Todo
}

func loadCompleteAllActiveTodosContext(ctx context.Context, readModel TodoReadModelReader, userRegisteredID string) (*completeAllActiveTodosContext, error) {
	todos, err := readModel.List(ctx, userRegisteredID)
	if err != nil {
		return nil, err
	}
	return &completeAllActiveTodosContext{todos: todos}, nil
}
