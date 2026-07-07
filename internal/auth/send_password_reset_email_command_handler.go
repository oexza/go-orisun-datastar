package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/commandlimits"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/protectedpii"
)

type SendPasswordResetEmailCommand struct {
	PasswordResetRequestedID string
	Metadata                 CommandMetadata
}

type passwordResetEmailContext struct {
	requestID   string
	email       string
	token       string
	expiresAt   string
	subjectKey  protectedpii.SubjectDataKey
	alreadySent bool
	position    eventstore.Position
	events      []eventstore.ResolvedEvent
	query       eventstore.Query
}

func SendPasswordResetEmailCommandHandler(ctx context.Context, command SendPasswordResetEmailCommand, saver eventstore.Saver, retriever eventstore.Retriever, sender EmailSender, appURL string, keys SubjectPiiKeyPort) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	requestedQuery := passwordResetRequestedQuery(command.PasswordResetRequestedID)
	sentQuery := passwordResetEmailSentQuery(command.PasswordResetRequestedID)
	query := combineQueries(requestedQuery, sentQuery)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return err
	}

	events := eventstore.EventsFromLatest(latest.Results)
	model := passwordResetEmailContext{
		position: latest.ContextPosition,
		events:   events,
		query:    query,
	}
	for _, resolved := range model.events {
		if resolved.Event.EventType == PasswordResetRequested {
			userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
			subjectKey, ok, err := keys.GetSubjectDataKey(ctx, userRegisteredID)
			if err != nil {
				return err
			}
			if !ok {
				return eventstore.ErrNotFound
			}
			model.subjectKey = subjectKey
			break
		}
	}
	for _, resolved := range model.events {
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

	id := uuidv7.NewString()
	sent := NewPasswordResetEmailSentEvent(id, time.Now(), command.PasswordResetRequestedID, nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{sent}, model.position, model.events, model.query)
	return err
}

func (m *passwordResetEmailContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case PasswordResetRequested:
		m.requestID, _ = resolved.Event.Data["passwordResetRequestedId"].(string)
		protector := protectedpii.FromEnv()
		m.email = protectedpii.MustDecryptEventStringWithDataKey(protector, m.subjectKey, resolved.Event.Data, PasswordResetRequestedEmailField)
		m.token = protectedpii.MustDecryptEventStringWithDataKey(protector, m.subjectKey, resolved.Event.Data, PasswordResetRequestedTokenField)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case PasswordResetEmailSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
