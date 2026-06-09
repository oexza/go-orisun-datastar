package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
)

type RegisterUserCommand struct {
	Username    string
	Email       string
	FirstName   string
	LastName    string
	YearOfBirth int
	Metadata    CommandMetadata
}

type RegisterUserResult struct {
	UserRegisteredID string
	Username         string
	Email            string
	FirstName        string
	LastName         string
}

func RegisterUserCommandHandler(ctx context.Context, command RegisterUserCommand, saver eventstore.Saver, retriever eventstore.Retriever) (RegisterUserResult, error) {
	model, err := loadRegisterUserContext(ctx, command, retriever)
	if err != nil {
		return RegisterUserResult{}, err
	}
	if model.existingUsername || model.existingEmail {
		return RegisterUserResult{}, errors.New("user already exists")
	}

	event := NewUserRegisteredEvent(model.userRegisteredID, model.username, model.email, model.firstName, model.lastName, command.YearOfBirth, metadataWithQuery(command.Metadata, model.query))
	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return RegisterUserResult{}, err
	}
	return RegisterUserResult{
		UserRegisteredID: model.userRegisteredID,
		Username:         model.username,
		Email:            model.email,
		FirstName:        model.firstName,
		LastName:         model.lastName,
	}, nil
}

type registerUserContext struct {
	existingUsername bool
	existingEmail    bool
	userRegisteredID string
	username         string
	email            string
	firstName        string
	lastName         string
	position         eventstore.Position
	events           []eventstore.ResolvedEvent
	query            eventstore.Query
}

func loadRegisterUserContext(ctx context.Context, command RegisterUserCommand, retriever eventstore.Retriever) (*registerUserContext, error) {
	username := strings.TrimSpace(command.Username)
	email := strings.ToLower(strings.TrimSpace(command.Email))
	if len(username) < 4 || email == "" {
		return nil, errors.New("invalid registration input")
	}

	query := userRegisteredByUsernameOrEmailQuery(username, email)
	events, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 2, eventstore.Forward, query)
	if err != nil {
		return nil, err
	}

	model := &registerUserContext{
		userRegisteredID: uuidv7.NewString(),
		username:         username,
		email:            email,
		firstName:        strings.TrimSpace(command.FirstName),
		lastName:         strings.TrimSpace(command.LastName),
		position:         eventstore.NoEventPosition,
		events:           events,
		query:            query,
	}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *registerUserContext) handle(resolved eventstore.ResolvedEvent) {
	if resolved.Event.EventType == UserRegistered {
		username, _ := resolved.Event.Data[UserRegisteredUsernameField].(string)
		email, _ := resolved.Event.Data[UserRegisteredEmailField].(string)
		if username == m.username {
			m.existingUsername = true
		}
		if email == m.email {
			m.existingEmail = true
		}
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
