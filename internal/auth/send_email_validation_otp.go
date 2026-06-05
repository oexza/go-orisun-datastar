package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type CommandMetadata = eventstore.CommandMetadata

type SendEmailValidationOTPCommand struct {
	UserRegisteredID                string
	EmailVerificationOTPGeneratedID string
	Metadata                        CommandMetadata
}

type emailValidationOTPContext struct {
	otpID       string
	code        string
	expiresAt   string
	email       string
	alreadySent bool
	position    eventstore.Position
}

func SendEmailValidationOTPCommandHandler(ctx context.Context, command SendEmailValidationOTPCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender EmailSender) error {
	generatedQuery := emailVerificationOTPGeneratedQuery(command.EmailVerificationOTPGeneratedID)
	generatedEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, generatedQuery)
	if err != nil {
		return err
	}

	userQuery := userRegisteredQuery(command.UserRegisteredID)
	userEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, userQuery)
	if err != nil {
		return err
	}

	sentQuery := emailVerificationOTPSentQuery(command.EmailVerificationOTPGeneratedID)
	sentEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, sentQuery)
	if err != nil {
		return err
	}

	model := emailValidationOTPContext{position: eventstore.NoEventPosition}
	for _, resolved := range append(append(generatedEvents, userEvents...), sentEvents...) {
		model.handle(resolved)
	}
	if model.alreadySent {
		return nil
	}
	if model.otpID == "" {
		return errors.New("email verification OTP generation event not found")
	}
	if model.email == "" {
		return errors.New("registered user event not found")
	}

	if err := sender.Send(ctx, email.Message{
		To:    model.email,
		Title: "Verify your email",
		Body:  "Your verification code is <strong>" + model.code + "</strong>.",
	}); err != nil {
		return err
	}

	id := uuidv7.NewString()
	sent := NewEmailVerificationOTPSentEvent(id, time.Now(), command.EmailVerificationOTPGeneratedID, metadataWithQuery(command.Metadata, combineQueries(generatedQuery, userQuery, sentQuery)))
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{sent}, model.position, append(generatedEvents, userEvents...), combineQueries(generatedQuery, userQuery, sentQuery))
	return err
}

func (m *emailValidationOTPContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case EmailVerificationOTPGenerated:
		m.otpID, _ = resolved.Event.Data["emailVerificationOTPGeneratedId"].(string)
		m.code, _ = resolved.Event.Data["otpCode"].(string)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case UserRegistered:
		m.email, _ = resolved.Event.Data["email"].(string)
	case EmailVerificationOTPSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
