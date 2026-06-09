package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type UpdateProfileBioCommand struct {
	User     views.User
	Bio      string
	Metadata map[string]any
}

func UpdateProfileBioCommandHandler(ctx context.Context, command UpdateProfileBioCommand, saver eventstore.Saver, retriever eventstore.Retriever) error {
	model, err := loadUpdateProfileBioContext(ctx, command, retriever)
	if err != nil {
		return err
	}
	if model.bio == model.nextBio {
		return nil
	}
	event := NewProfileBioUpdatedEvent(model.eventID, model.nextBio, time.Now(), command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	_, err = saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type updateProfileBioContext struct {
	userExists bool
	bio        string
	nextBio    string
	eventID    string
	position   eventstore.Position
	events     []eventstore.ResolvedEvent
	query      eventstore.Query
}

func loadUpdateProfileBioContext(ctx context.Context, command UpdateProfileBioCommand, retriever eventstore.Retriever) (*updateProfileBioContext, error) {
	bio := strings.TrimSpace(command.Bio)
	if len(bio) > 280 {
		return nil, errors.New("bio must be 280 characters or fewer")
	}
	userQuery := registeredUserQuery(command.User.UserRegisteredID)
	bioQuery := profileUserEventQuery(ProfileBioUpdated, command.User.UserRegisteredID)
	query := combineQueries(userQuery, bioQuery)
	model := &updateProfileBioContext{
		nextBio:  bio,
		eventID:  uuidv7.NewString(),
		position: eventstore.NoEventPosition,
		query:    query,
	}
	userEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, userQuery)
	if err != nil {
		return nil, err
	}
	bioEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, bioQuery)
	if err != nil {
		return nil, err
	}
	model.events = append(append([]eventstore.ResolvedEvent{}, userEvents...), bioEvents...)
	for _, event := range model.events {
		model.handle(event)
	}
	if !model.userExists {
		return nil, eventstore.ErrNotFound
	}
	return model, nil
}

func (m *updateProfileBioContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case "UserRegistered":
		m.userExists = true
	case ProfileBioUpdated:
		m.bio, _ = resolved.Event.Data[ProfileBioUpdatedBioField].(string)
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
