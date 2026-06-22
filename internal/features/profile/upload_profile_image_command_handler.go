package profile

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type UploadProfileImageCommand struct {
	User        views.User
	Data        []byte
	ContentType string
	Header      bool
	Metadata    eventstore.CommandMetadata
}

type UploadProfileImageResult struct {
	URL string
}

func UploadProfileImageCommandHandler(ctx context.Context, command UploadProfileImageCommand, saver eventstore.Saver, retriever eventstore.Retriever, storage ObjectStore) (UploadProfileImageResult, error) {
	model, err := loadUploadProfileImageContext(ctx, command, retriever)
	if err != nil {
		return UploadProfileImageResult{}, err
	}
	if err := storage.PutObject(ctx, model.key, command.Data, command.ContentType); err != nil {
		return UploadProfileImageResult{}, err
	}
	url := storage.PublicURL(model.key)
	event := NewProfileImageUploadedEvent(model.eventID, url, time.Now(), command.User.UserRegisteredID, nil)
	if command.Header {
		event = NewProfileHeaderImageUploadedEvent(model.eventID, url, time.Now(), command.User.UserRegisteredID, nil)
	}
	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return UploadProfileImageResult{}, err
	}
	return UploadProfileImageResult{URL: url}, nil
}

type uploadProfileImageContext struct {
	userExists bool
	key        string
	eventID    string
	position   eventstore.Position
	events     []eventstore.ResolvedEvent
	query      eventstore.Query
}

func loadUploadProfileImageContext(ctx context.Context, command UploadProfileImageCommand, retriever eventstore.Retriever) (*uploadProfileImageContext, error) {
	if len(command.Data) == 0 {
		return nil, errors.New("missing image")
	}
	if len(command.Data) > 5*1024*1024 {
		return nil, errors.New("image must be 5MB or smaller")
	}
	ext := extension(command.ContentType)
	kind := "avatar"
	eventType := ProfileImageUploaded
	if command.Header {
		kind = "header"
		eventType = ProfileHeaderImageUploaded
	}
	userQuery := registeredUserQuery(command.User.UserRegisteredID)
	imageQuery := profileUserEventQuery(eventType, command.User.UserRegisteredID)
	query := combineQueries(userQuery, imageQuery)
	key := filepath.ToSlash(fmt.Sprintf("profiles/%s/%s-%s.%s", command.User.UserRegisteredID, kind, uuidv7.NewString(), ext))
	eventID := uuidv7.NewString()
	model := &uploadProfileImageContext{
		key:      key,
		eventID:  eventID,
		position: eventstore.NoEventPosition,
		query:    query,
	}
	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}
	model.events = eventstore.EventsFromLatest(latest.Results)
	model.position = latest.ContextPosition
	for _, event := range model.events {
		model.handle(event)
	}
	if !model.userExists {
		return nil, eventstore.ErrNotFound
	}
	return model, nil
}

func (m *uploadProfileImageContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case "UserRegistered":
		m.userExists = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
