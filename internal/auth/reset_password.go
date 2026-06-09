package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type ResetPasswordCommand struct {
	User                     views.User
	PasswordResetRequestedID string
	Metadata                 CommandMetadata
}

func ResetPasswordCommandHandler(ctx context.Context, command ResetPasswordCommand, saver eventstore.Saver, retriever eventstore.Retriever) error {
	model, err := loadResetPasswordContext(ctx, command, retriever)
	if err != nil {
		return err
	}
	if !model.requestExists {
		return errors.New("password reset request event not found")
	}
	if model.alreadyCompleted {
		return errors.New("password reset already completed")
	}

	eventID := uuidv7.NewString()
	event := NewPasswordResetCompletedEvent(eventID, time.Now(), command.PasswordResetRequestedID, command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type resetPasswordContext struct {
	requestExists    bool
	alreadyCompleted bool
	position         eventstore.Position
	events           []eventstore.ResolvedEvent
	query            eventstore.Query
}

func loadResetPasswordContext(ctx context.Context, command ResetPasswordCommand, retriever eventstore.Retriever) (*resetPasswordContext, error) {
	requestedQuery := passwordResetRequestedQuery(command.PasswordResetRequestedID)
	completedQuery := passwordResetCompletedQuery(command.PasswordResetRequestedID)
	query := combineQueries(requestedQuery, completedQuery)
	requestedEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, requestedQuery)
	if err != nil {
		return nil, err
	}
	completedEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, completedQuery)
	if err != nil {
		return nil, err
	}
	events := append(append([]eventstore.ResolvedEvent{}, requestedEvents...), completedEvents...)
	model := &resetPasswordContext{position: eventstore.NoEventPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *resetPasswordContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case PasswordResetRequested:
		m.requestExists = true
	case PasswordResetCompleted:
		m.alreadyCompleted = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
