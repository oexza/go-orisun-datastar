package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/commandlimits"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/protectedpii"
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
	subjectKey  protectedpii.SubjectDataKey
	alreadySent bool
	position    eventstore.Position
	events      []eventstore.ResolvedEvent
	query       eventstore.Query
}

func SendEmailValidationOTPCommandHandler(ctx context.Context, command SendEmailValidationOTPCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender EmailSender, keys SubjectPiiKeyPort) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	generatedQuery := emailVerificationOTPGeneratedQuery(command.EmailVerificationOTPGeneratedID)
	userQuery := userRegisteredQuery(command.UserRegisteredID)
	sentQuery := emailVerificationOTPSentQuery(command.EmailVerificationOTPGeneratedID)
	query := combineQueries(generatedQuery, userQuery, sentQuery)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return err
	}

	events := eventstore.EventsFromLatest(latest.Results)
	model := emailValidationOTPContext{
		position: latest.ContextPosition,
		events:   events,
		query:    query,
	}
	subjectKey, ok, err := keys.GetSubjectDataKey(ctx, command.UserRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	model.subjectKey = subjectKey
	for _, resolved := range model.events {
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
	sent := NewEmailVerificationOTPSentEvent(id, time.Now(), command.EmailVerificationOTPGeneratedID, nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{sent}, model.position, model.events, model.query)
	return err
}

func (m *emailValidationOTPContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case EmailVerificationOTPGenerated:
		m.otpID, _ = resolved.Event.Data["emailVerificationOTPGeneratedId"].(string)
		m.code, _ = resolved.Event.Data["otpCode"].(string)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case UserRegistered:
		m.email = protectedpii.MustDecryptEventStringWithDataKey(protectedpii.FromEnv(), m.subjectKey, resolved.Event.Data, UserRegisteredEmailField)
	case EmailVerificationOTPSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
