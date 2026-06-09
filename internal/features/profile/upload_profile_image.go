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
	Metadata    map[string]any
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
	event := NewProfileImageUploadedEvent(model.eventID, url, time.Now(), command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	if command.Header {
		event = NewProfileHeaderImageUploadedEvent(model.eventID, url, time.Now(), command.User.UserRegisteredID, metadataWithQuery(command.Metadata, model.query))
	}
	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
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
	userEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, userQuery)
	if err != nil {
		return nil, err
	}
	imageEvents, err := retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, imageQuery)
	if err != nil {
		return nil, err
	}
	model.events = append(append([]eventstore.ResolvedEvent{}, userEvents...), imageEvents...)
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
