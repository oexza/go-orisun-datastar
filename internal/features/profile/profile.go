package profile

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/storage"
	"github.com/example/hono-event-starter-go/internal/views"
)

type Service struct {
	store   eventstore.Saver
	storage storage.Provider
}

func NewService(store eventstore.Saver, storage storage.Provider) *Service {
	return &Service{store: store, storage: storage}
}

func (s *Service) UpdateBio(ctx context.Context, user views.User, bio string) error {
	return s.UpdateBioWithMetadata(ctx, user, bio, nil)
}

func (s *Service) UpdateBioWithMetadata(ctx context.Context, user views.User, bio string, metadata map[string]any) error {
	bio = strings.TrimSpace(bio)
	if len(bio) > 280 {
		return errors.New("bio must be 280 characters or fewer")
	}
	eventID := uuid.NewString()
	event := NewProfileBioUpdatedEvent(eventID, bio, time.Now(), user.UserRegisteredID, metadata)
	_, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, profileEventQuery(ProfileBioUpdated, ProfileBioUpdatedIDField, eventID))
	return err
}

func (s *Service) UploadImage(ctx context.Context, user views.User, data []byte, contentType string, header bool) (string, error) {
	return s.UploadImageWithMetadata(ctx, user, data, contentType, header, nil)
}

func (s *Service) UploadImageWithMetadata(ctx context.Context, user views.User, data []byte, contentType string, header bool, metadata map[string]any) (string, error) {
	if len(data) == 0 {
		return "", errors.New("missing image")
	}
	if len(data) > 5*1024*1024 {
		return "", errors.New("image must be 5MB or smaller")
	}
	ext := extension(contentType)
	kind := "avatar"
	eventType := ProfileImageUploaded
	if header {
		kind = "header"
		eventType = ProfileHeaderImageUploaded
	}
	key := filepath.ToSlash(fmt.Sprintf("profiles/%s/%s-%s.%s", user.UserRegisteredID, kind, uuid.NewString(), ext))
	if err := s.storage.PutObject(ctx, key, data, contentType); err != nil {
		return "", err
	}
	url := s.storage.PublicURL(key)
	eventID := uuid.NewString()
	idField := ProfileImageUploadedIDField
	event := NewProfileImageUploadedEvent(eventID, url, time.Now(), user.UserRegisteredID, metadata)
	if header {
		idField = ProfileHeaderImageUploadedIDField
		event = NewProfileHeaderImageUploadedEvent(eventID, url, time.Now(), user.UserRegisteredID, metadata)
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, profileEventQuery(eventType, idField, eventID)); err != nil {
		return "", err
	}
	return url, nil
}

func extension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func profileEventQuery(eventType, idField, id string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: eventType},
		{Key: idField, Value: id},
	}}}}
}
