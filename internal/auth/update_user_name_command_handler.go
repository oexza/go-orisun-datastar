package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/protectedpii"
	"github.com/OrisunLabs/go-orisun-datastar/internal/uuidv7"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
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

func UpdateUserNameCommandHandler(ctx context.Context, command UpdateUserNameCommand, saver eventstore.Saver, retriever eventstore.Retriever, keys SubjectPiiKeyPort) (UpdateUserNameResult, error) {
	if err := commandlimits.Assert(command); err != nil {
		return UpdateUserNameResult{}, err
	}
	model, err := loadUpdateUserNameContext(ctx, command, retriever, keys)
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
	event := NewUserNameChangedEvent(eventID, model.nextName, time.Now(), command.User.UserRegisteredID, model.subjectKey, nil)
	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return UpdateUserNameResult{}, err
	}
	return UpdateUserNameResult{Name: model.nextName}, nil
}

type updateUserNameContext struct {
	userExists bool
	name       string
	nextName   string
	subjectKey protectedpii.SubjectDataKey
	position   eventstore.Position
	events     []eventstore.ResolvedEvent
	query      eventstore.Query
}

func loadUpdateUserNameContext(ctx context.Context, command UpdateUserNameCommand, retriever eventstore.Retriever, keys SubjectPiiKeyPort) (*updateUserNameContext, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	userQuery := userRegisteredQuery(command.User.UserRegisteredID)
	nameQuery := userNameChangedByUserQuery(command.User.UserRegisteredID)
	query := combineQueries(userQuery, nameQuery)
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	events := eventstore.EventsFromLatest(latest.Results)
	subjectKey, ok, err := keys.GetSubjectDataKey(ctx, command.User.UserRegisteredID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, eventstore.ErrNotFound
	}
	model := &updateUserNameContext{nextName: name, subjectKey: subjectKey, position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *updateUserNameContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case UserRegistered:
		m.userExists = true
		protector := protectedpii.FromEnv()
		firstName := protectedpii.MustDecryptEventStringWithDataKey(protector, m.subjectKey, resolved.Event.Data, UserRegisteredFirstNameField)
		lastName := protectedpii.MustDecryptEventStringWithDataKey(protector, m.subjectKey, resolved.Event.Data, UserRegisteredLastNameField)
		username := protectedpii.MustDecryptEventStringWithDataKey(protector, m.subjectKey, resolved.Event.Data, UserRegisteredUsernameField)
		m.name = strings.TrimSpace(firstName + " " + lastName)
		if m.name == "" {
			m.name = username
		}
	case UserNameChanged:
		m.name = protectedpii.MustDecryptEventStringWithDataKey(protectedpii.FromEnv(), m.subjectKey, resolved.Event.Data, UserNameChangedNameField)
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
