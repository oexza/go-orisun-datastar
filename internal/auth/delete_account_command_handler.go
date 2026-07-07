package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/commandlimits"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"golang.org/x/crypto/bcrypt"
)

type RequestAccountDeletionCommand struct {
	User     views.User
	Password string
	Metadata CommandMetadata
}

type CompleteAccountDeletionCommand struct {
	AccountDeletionRequestedID string
	UserRegisteredID           string
	AuthUserID                 string
	Metadata                   CommandMetadata
}

type AccountDataDeletionPort interface {
	DeleteAccountData(ctx context.Context, userRegisteredID string) error
}

type AccountPiiKeyPort interface {
	DestroySubjectKey(ctx context.Context, userRegisteredID string) error
}

func RequestAccountDeletionCommandHandler(ctx context.Context, command RequestAccountDeletionCommand, credentials PasswordCredentialReader, saver eventstore.Saver, retriever eventstore.Retriever) (string, error) {
	if err := commandlimits.Assert(command); err != nil {
		return "", err
	}
	if command.User.UserRegisteredID == "" {
		return "", errors.New("user is required")
	}
	_, hash, err := credentials.UserByEmailWithPassword(ctx, command.User.Email)
	if err != nil {
		return "", errors.New("invalid password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(command.Password)); err != nil {
		return "", errors.New("invalid password")
	}

	model, err := loadAccountDeletionRequestContext(ctx, command.User.UserRegisteredID, retriever)
	if err != nil {
		return "", err
	}
	if model.deleted || model.requestedID != "" {
		return model.requestedID, nil
	}
	requestID := uuidv7.NewString()
	event := NewAccountDeletionRequestedEvent(requestID, time.Now(), command.User.ID, command.User.UserRegisteredID, nil)
	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return "", err
	}
	return requestID, nil
}

func CompleteAccountDeletionCommandHandler(ctx context.Context, command CompleteAccountDeletionCommand, data AccountDataDeletionPort, keys AccountPiiKeyPort, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	model, err := loadAccountDeletionCompletionContext(ctx, command.AccountDeletionRequestedID, command.UserRegisteredID, retriever)
	if err != nil {
		return err
	}
	if !model.requested {
		return errors.New("account deletion request event not found")
	}
	if model.deleted {
		return nil
	}
	if err := data.DeleteAccountData(ctx, command.UserRegisteredID); err != nil {
		return err
	}
	if err := keys.DestroySubjectKey(ctx, command.UserRegisteredID); err != nil {
		return err
	}
	deletedID := uuidv7.NewString()
	event := NewAccountDeletedEvent(deletedID, time.Now(), command.AccountDeletionRequestedID, command.AuthUserID, command.UserRegisteredID, nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type accountDeletionRequestContext struct {
	requestedID string
	deleted     bool
	position    eventstore.Position
	events      []eventstore.ResolvedEvent
	query       eventstore.Query
}

func loadAccountDeletionRequestContext(ctx context.Context, userRegisteredID string, retriever eventstore.Retriever) (*accountDeletionRequestContext, error) {
	query := combineQueries(userRegisteredQuery(userRegisteredID), accountDeletionRequestedByUserQuery(userRegisteredID), accountDeletedByUserQuery(userRegisteredID))
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	model := &accountDeletionRequestContext{position: latest.ContextPosition, events: eventstore.EventsFromLatest(latest.Results), query: query}
	for _, resolved := range model.events {
		switch resolved.Event.EventType {
		case AccountDeletionRequested:
			model.requestedID, _ = resolved.Event.Data[AccountDeletionRequestedIDField].(string)
		case AccountDeleted:
			model.deleted = true
		}
	}
	return model, nil
}

type accountDeletionCompletionContext struct {
	requested bool
	deleted   bool
	position  eventstore.Position
	events    []eventstore.ResolvedEvent
	query     eventstore.Query
}

func loadAccountDeletionCompletionContext(ctx context.Context, requestID, userRegisteredID string, retriever eventstore.Retriever) (*accountDeletionCompletionContext, error) {
	query := combineQueries(accountDeletionRequestedByUserQuery(userRegisteredID), accountDeletedByRequestQuery(requestID))
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	model := &accountDeletionCompletionContext{position: latest.ContextPosition, events: eventstore.EventsFromLatest(latest.Results), query: query}
	for _, resolved := range model.events {
		switch resolved.Event.EventType {
		case AccountDeletionRequested:
			id, _ := resolved.Event.Data[AccountDeletionRequestedIDField].(string)
			model.requested = id == requestID
		case AccountDeleted:
			model.deleted = true
		}
	}
	return model, nil
}
