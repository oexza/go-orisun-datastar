package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type CompleteAllActiveTodosCommand struct {
	UserRegisteredID string
	Metadata         CommandMetadata
}

func CompleteAllActiveTodosCommandHandler(ctx context.Context, command CompleteAllActiveTodosCommand, readModel TodoReadModelReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	todos, err := readModel.List(ctx, command.UserRegisteredID)
	if err != nil {
		return err
	}
	for _, item := range todos {
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
