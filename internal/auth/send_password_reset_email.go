package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type SendPasswordResetEmailCommand struct {
	PasswordResetRequestedID string
	Metadata                 CommandMetadata
}

func SendPasswordResetEmailCommandHandler(ctx context.Context, command SendPasswordResetEmailCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender EmailSender, appURL string) error {
	requestedQuery := passwordResetRequestedQuery(command.PasswordResetRequestedID)
	requestedEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, requestedQuery)
	if err != nil {
		return err
	}

	sentQuery := passwordResetEmailSentQuery(command.PasswordResetRequestedID)
	sentEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, sentQuery)
	if err != nil {
		return err
	}

	model := passwordResetEmailContext{position: eventstore.NoEventPosition}
	for _, resolved := range append(requestedEvents, sentEvents...) {
		model.handle(resolved)
	}
	if model.alreadySent {
		return nil
	}
	if model.requestID == "" {
		return errors.New("password reset request event not found")
	}

	resetURL := fmt.Sprintf("%s/reset-password/%s", appURL, model.token)
	if err := sender.Send(ctx, email.Message{
		To:    model.email,
		Title: "Reset your password",
		Body:  fmt.Sprintf(`Click the link below to reset your password. This link expires at %s.<br><br><a href="%s">%s</a>`, model.expiresAt, resetURL, resetURL),
	}); err != nil {
		return err
	}

	id := uuid.NewString()
	sent := NewPasswordResetEmailSentEvent(id, time.Now(), command.PasswordResetRequestedID, metadataWithQuery(command.Metadata, combineQueries(requestedQuery, sentQuery)))
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{sent}, model.position, requestedEvents, combineQueries(requestedQuery, sentQuery))
	return err
}
