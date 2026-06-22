package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"golang.org/x/crypto/bcrypt"
)

type ResetPasswordCommand struct {
	Token    string
	Password string
	Metadata CommandMetadata
}

type PasswordResetReader interface {
	PasswordResetByToken(ctx context.Context, token string) (PasswordResetVerification, error)
}

func ResetPasswordCommandHandler(ctx context.Context, command ResetPasswordCommand, resets PasswordResetReader, users AuthUserByIDReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if len(command.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	verification, err := resets.PasswordResetByToken(ctx, command.Token)
	if err != nil {
		return err
	}
	user, err := users.UserByIDOrRegisteredID(ctx, verification.UserID)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(command.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	model, err := loadResetPasswordContext(ctx, verification.ID, retriever)
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
	event := NewPasswordResetCompletedEvent(eventID, time.Now(), verification.ID, user.UserRegisteredID, string(hash), nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type resetPasswordContext struct {
	requestExists    bool
	alreadyCompleted bool
	position         eventstore.Position
	events           []eventstore.ResolvedEvent
	query            eventstore.Query
}

func loadResetPasswordContext(ctx context.Context, passwordResetRequestedID string, retriever eventstore.Retriever) (*resetPasswordContext, error) {
	requestedQuery := passwordResetRequestedQuery(passwordResetRequestedID)
	completedQuery := passwordResetCompletedQuery(passwordResetRequestedID)
	query := combineQueries(requestedQuery, completedQuery)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)
	model := &resetPasswordContext{position: latest.ContextPosition, events: events, query: query}
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
