package todo

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

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
