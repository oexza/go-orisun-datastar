package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type ReopenAllCompletedTodosCommand struct {
	UserRegisteredID string
	Metadata         CommandMetadata
}

func ReopenAllCompletedTodosCommandHandler(ctx context.Context, command ReopenAllCompletedTodosCommand, readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	todos, err := readModel.List(ctx, command.UserRegisteredID)
	if err != nil {
		return err
	}
	for _, item := range todos {
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
