package auth

import (
	"context"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type RequestPasswordResetCommand struct {
	User     views.User
	Metadata CommandMetadata
}

type RequestPasswordResetResult struct {
	PasswordResetRequestedID string
	Token                    string
	ExpiresAt                time.Time
}

func RequestPasswordResetCommandHandler(ctx context.Context, command RequestPasswordResetCommand, saver eventstore.Saver, retriever eventstore.Retriever) (RequestPasswordResetResult, error) {
	model, err := loadRequestPasswordResetContext(ctx, command, retriever)
	if err != nil {
		return RequestPasswordResetResult{}, err
	}

	token, err := randomToken(32)
	if err != nil {
		return RequestPasswordResetResult{}, err
	}
	requestID := uuidv7.NewString()
	expiresAt := time.Now().Add(30 * time.Minute)
	event := NewPasswordResetRequestedEvent(requestID, command.User.Email, token, expiresAt, command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return RequestPasswordResetResult{}, err
	}
	return RequestPasswordResetResult{PasswordResetRequestedID: requestID, Token: token, ExpiresAt: expiresAt}, nil
}

type requestPasswordResetContext struct {
	position eventstore.Position
	events   []eventstore.ResolvedEvent
	query    eventstore.Query
}

func loadRequestPasswordResetContext(ctx context.Context, command RequestPasswordResetCommand, retriever eventstore.Retriever) (*requestPasswordResetContext, error) {
	query := userRegisteredQuery(command.User.UserRegisteredID)
	events, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, query)
	if err != nil {
		return nil, err
	}
	model := &requestPasswordResetContext{position: eventstore.NoEventPosition, events: events, query: query}
	for _, event := range events {
		if event.Position.After(model.position) {
			model.position = event.Position
		}
	}
	return model, nil
}
