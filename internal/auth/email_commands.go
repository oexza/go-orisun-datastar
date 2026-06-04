package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/example/hono-event-starter-go/internal/email"
	"github.com/example/hono-event-starter-go/internal/eventstore"
)

type CommandMetadata = eventstore.CommandMetadata

type SendEmailValidationOTPCommand struct {
	UserRegisteredID                string
	EmailVerificationOTPGeneratedID string
	Metadata                        CommandMetadata
}

func SendEmailValidationOTPCommandHandler(ctx context.Context, command SendEmailValidationOTPCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender email.Sender) error {
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

	id := uuid.NewString()
	sent := eventstore.DomainEvent{
		EventID:   id,
		EventType: EmailVerificationOTPSent,
		Data: map[string]any{
			"emailVerificationOTPSentId": id,
			"sentAt":                     time.Now().Format(time.RFC3339),
			"scope": map[string]any{
				"emailVerificationOTPGeneratedId": command.EmailVerificationOTPGeneratedID,
			},
		},
		Metadata: metadataWithQuery(command.Metadata, combineQueries(generatedQuery, userQuery, sentQuery)),
	}
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{sent}, model.position, append(generatedEvents, userEvents...), combineQueries(generatedQuery, userQuery, sentQuery))
	return err
}

type SendPasswordResetEmailCommand struct {
	PasswordResetRequestedID string
	Metadata                 CommandMetadata
}

func SendPasswordResetEmailCommandHandler(ctx context.Context, command SendPasswordResetEmailCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender email.Sender, appURL string) error {
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
	sent := eventstore.DomainEvent{
		EventID:   id,
		EventType: PasswordResetEmailSent,
		Data: map[string]any{
			"passwordResetEmailSentId": id,
			"sentAt":                   time.Now().Format(time.RFC3339),
			"scope": map[string]any{
				"passwordResetRequestedId": command.PasswordResetRequestedID,
			},
		},
		Metadata: metadataWithQuery(command.Metadata, combineQueries(requestedQuery, sentQuery)),
	}
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{sent}, model.position, requestedEvents, combineQueries(requestedQuery, sentQuery))
	return err
}

type emailValidationOTPContext struct {
	otpID       string
	code        string
	expiresAt   string
	email       string
	alreadySent bool
	position    eventstore.Position
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

type passwordResetEmailContext struct {
	requestID   string
	email       string
	token       string
	expiresAt   string
	alreadySent bool
	position    eventstore.Position
}

func (m *passwordResetEmailContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case PasswordResetRequested:
		m.requestID, _ = resolved.Event.Data["passwordResetRequestedId"].(string)
		m.email, _ = resolved.Event.Data["email"].(string)
		m.token, _ = resolved.Event.Data["resetToken"].(string)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case PasswordResetEmailSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}

func userRegisteredQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: UserRegistered},
		{Key: "userRegisteredId", Value: userRegisteredID},
	}}}}
}

func emailVerificationOTPGeneratedQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: "emailVerificationOTPGeneratedId", Value: otpID},
	}}}}
}

func emailVerificationOTPSentQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPSent},
		{Key: "scope.emailVerificationOTPGeneratedId", Value: otpID},
	}}}}
}

func emailVerificationOTPValidatedQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPValidated},
		{Key: "scope.emailVerificationOTPGeneratedId", Value: otpID},
	}}}}
}

func passwordResetRequestedQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordResetRequested},
		{Key: "passwordResetRequestedId", Value: requestID},
	}}}}
}

func passwordResetEmailSentQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordResetEmailSent},
		{Key: "scope.passwordResetRequestedId", Value: requestID},
	}}}}
}

func combineQueries(queries ...eventstore.Query) eventstore.Query {
	combined := eventstore.Query{}
	for _, query := range queries {
		combined.Criteria = append(combined.Criteria, query.Criteria...)
	}
	return combined
}

func metadataWithQuery(metadata CommandMetadata, query eventstore.Query) map[string]any {
	return eventstore.MergeMetadata(map[string]any{"query": eventstore.MustJSON(query)}, metadata)
}
