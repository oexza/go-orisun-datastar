package auth

import (
	"context"
	"errors"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type ChangePasswordCommand struct {
	User     views.User
	Metadata CommandMetadata
}

func ChangePasswordCommandHandler(ctx context.Context, command ChangePasswordCommand, saver eventstore.Saver, retriever eventstore.Retriever) error {
	model, err := loadChangePasswordContext(ctx, command, retriever)
	if err != nil {
		return err
	}
	if !model.userExists {
		return errors.New("registered user event not found")
	}

	eventID := uuidv7.NewString()
	event := NewPasswordChangedEvent(eventID, time.Now(), command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
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
	userEvents, err := retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, userQuery)
	if err != nil {
		return nil, err
	}
	passwordEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, passwordQuery)
	if err != nil {
		return nil, err
	}
	events := append(append([]eventstore.ResolvedEvent{}, userEvents...), passwordEvents...)
	model := &changePasswordContext{position: eventstore.NoEventPosition, events: events, query: query}
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
