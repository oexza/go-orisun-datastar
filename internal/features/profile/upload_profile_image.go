package profile

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

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

func UploadProfileImageCommandHandler(ctx context.Context, command UploadProfileImageCommand, saver eventstore.Saver, storage ObjectStore) (UploadProfileImageResult, error) {
	if len(command.Data) == 0 {
		return UploadProfileImageResult{}, errors.New("missing image")
	}
	if len(command.Data) > 5*1024*1024 {
		return UploadProfileImageResult{}, errors.New("image must be 5MB or smaller")
	}
	ext := extension(command.ContentType)
	kind := "avatar"
	eventType := ProfileImageUploaded
	if command.Header {
		kind = "header"
		eventType = ProfileHeaderImageUploaded
	}
	key := filepath.ToSlash(fmt.Sprintf("profiles/%s/%s-%s.%s", command.User.UserRegisteredID, kind, uuid.NewString(), ext))
	if err := storage.PutObject(ctx, key, command.Data, command.ContentType); err != nil {
		return UploadProfileImageResult{}, err
	}
	url := storage.PublicURL(key)
	eventID := uuid.NewString()
	idField := ProfileImageUploadedIDField
	event := NewProfileImageUploadedEvent(eventID, url, time.Now(), command.User.UserRegisteredID, command.Metadata)
	if command.Header {
		idField = ProfileHeaderImageUploadedIDField
		event = NewProfileHeaderImageUploadedEvent(eventID, url, time.Now(), command.User.UserRegisteredID, command.Metadata)
	}
	if _, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, profileEventQuery(eventType, idField, eventID)); err != nil {
		return UploadProfileImageResult{}, err
	}
	return UploadProfileImageResult{URL: url}, nil
}
