package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

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
