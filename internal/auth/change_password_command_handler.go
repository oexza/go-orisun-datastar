package auth

import (
	"context"
	"errors"
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/uuidv7"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
	"golang.org/x/crypto/bcrypt"
)

type ChangePasswordCommand struct {
	User            views.User
	CurrentPassword string
	NewPassword     string
	Metadata        CommandMetadata
}

type PasswordCredentialReader interface {
	UserByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error)
}

func ChangePasswordCommandHandler(ctx context.Context, command ChangePasswordCommand, credentials PasswordCredentialReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	if len(command.NewPassword) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	_, currentHash, err := credentials.UserByEmailWithPassword(ctx, command.User.Email)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(command.CurrentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(command.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	model, err := loadChangePasswordContext(ctx, command, retriever)
	if err != nil {
		return err
	}
	if !model.userExists {
		return errors.New("registered user event not found")
	}

	eventID := uuidv7.NewString()
	event := NewPasswordChangedEvent(eventID, time.Now(), command.User.UserRegisteredID, string(newHash), nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type changePasswordContext struct {
	userExists bool
	position   eventstore.Position
	events     []eventstore.ResolvedEvent
	query      eventstore.Query
}

func loadChangePasswordContext(ctx context.Context, command ChangePasswordCommand, retriever eventstore.Retriever) (*changePasswordContext, error) {
	userQuery := userRegisteredQuery(command.User.UserRegisteredID)
	passwordQuery := passwordChangedByUserQuery(command.User.UserRegisteredID)
	query := combineQueries(userQuery, passwordQuery)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)
	model := &changePasswordContext{position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *changePasswordContext) handle(resolved eventstore.ResolvedEvent) {
	if resolved.Event.EventType == UserRegistered {
		m.userExists = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
