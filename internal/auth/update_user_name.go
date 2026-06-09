package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type UpdateUserNameCommand struct {
	User     views.User
	Name     string
	Metadata CommandMetadata
}

type UpdateUserNameResult struct {
	Name    string
	Skipped bool
}

func UpdateUserNameCommandHandler(ctx context.Context, command UpdateUserNameCommand, saver eventstore.Saver, retriever eventstore.Retriever) (UpdateUserNameResult, error) {
	model, err := loadUpdateUserNameContext(ctx, command, retriever)
	if err != nil {
		return UpdateUserNameResult{}, err
	}
	if !model.userExists {
		return UpdateUserNameResult{}, errors.New("registered user event not found")
	}
	if model.name == model.nextName {
		return UpdateUserNameResult{Name: model.nextName, Skipped: true}, nil
	}

	eventID := uuidv7.NewString()
	event := NewUserNameChangedEvent(eventID, model.nextName, time.Now(), command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return UpdateUserNameResult{}, err
	}
	return UpdateUserNameResult{Name: model.nextName}, nil
}

type updateUserNameContext struct {
	userExists bool
	name       string
	nextName   string
	position   eventstore.Position
	events     []eventstore.ResolvedEvent
	query      eventstore.Query
}

func loadUpdateUserNameContext(ctx context.Context, command UpdateUserNameCommand, retriever eventstore.Retriever) (*updateUserNameContext, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	userQuery := userRegisteredQuery(command.User.UserRegisteredID)
	nameQuery := userNameChangedByUserQuery(command.User.UserRegisteredID)
	query := combineQueries(userQuery, nameQuery)
	userEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, userQuery)
	if err != nil {
		return nil, err
	}
	nameEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, nameQuery)
	if err != nil {
		return nil, err
	}
	events := append(append([]eventstore.ResolvedEvent{}, userEvents...), nameEvents...)
	model := &updateUserNameContext{nextName: name, position: eventstore.NoEventPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *updateUserNameContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case UserRegistered:
		m.userExists = true
		firstName, _ := resolved.Event.Data[UserRegisteredFirstNameField].(string)
		lastName, _ := resolved.Event.Data[UserRegisteredLastNameField].(string)
		username, _ := resolved.Event.Data[UserRegisteredUsernameField].(string)
		m.name = strings.TrimSpace(firstName + " " + lastName)
		if m.name == "" {
			m.name = username
		}
	case UserNameChanged:
		m.name, _ = resolved.Event.Data[UserNameChangedNameField].(string)
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
