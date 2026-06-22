package auth

import (
	"context"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type RequestPasswordResetCommand struct {
	EmailAddress string
	Metadata     CommandMetadata
}

type RequestPasswordResetResult struct {
	PasswordResetRequestedID string
	Token                    string
	ExpiresAt                time.Time
}

type PasswordResetUserReader interface {
	UserByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error)
}

func RequestPasswordResetCommandHandler(ctx context.Context, command RequestPasswordResetCommand, users PasswordResetUserReader, saver eventstore.Saver, retriever eventstore.Retriever) (RequestPasswordResetResult, error) {
	user, _, err := users.UserByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(command.EmailAddress)))
	if err != nil {
		return RequestPasswordResetResult{}, nil
	}
	return requestPasswordReset(ctx, user, command.Metadata, saver, retriever)
}

func requestPasswordReset(ctx context.Context, user views.User, metadata CommandMetadata, saver eventstore.Saver, retriever eventstore.Retriever) (RequestPasswordResetResult, error) {
	model, err := loadRequestPasswordResetContext(ctx, user, retriever)
	if err != nil {
		return RequestPasswordResetResult{}, err
	}

	token, err := randomToken(32)
	if err != nil {
		return RequestPasswordResetResult{}, err
	}
	requestID := uuidv7.NewString()
	expiresAt := time.Now().Add(30 * time.Minute)
	event := NewPasswordResetRequestedEvent(requestID, user.Email, token, expiresAt, user.UserRegisteredID, nil)
	if _, err := eventstore.SaveCommandEvents(ctx, saver, metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return RequestPasswordResetResult{}, err
	}
	return RequestPasswordResetResult{PasswordResetRequestedID: requestID, Token: token, ExpiresAt: expiresAt}, nil
}

type requestPasswordResetContext struct {
	position eventstore.Position
	events   []eventstore.ResolvedEvent
	query    eventstore.Query
}

func loadRequestPasswordResetContext(ctx context.Context, user views.User, retriever eventstore.Retriever) (*requestPasswordResetContext, error) {
	query := userRegisteredQuery(user.UserRegisteredID)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)
	model := &requestPasswordResetContext{position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		if event.Position.After(model.position) {
			model.position = event.Position
		}
	}
	return model, nil
}
