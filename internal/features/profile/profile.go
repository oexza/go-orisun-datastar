package profile

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/storage"
	"github.com/example/hono-event-starter-go/internal/views"
)

const (
	ProfileBioUpdated          = "ProfileBioUpdated"
	ProfileImageUploaded       = "ProfileImageUploaded"
	ProfileHeaderImageUploaded = "ProfileHeaderImageUploaded"
)

type Service struct {
	db      *appdb.DB
	store   eventstore.Saver
	storage storage.Provider
}

func NewService(db *appdb.DB, store eventstore.Saver, storage storage.Provider) *Service {
	return &Service{db: db, store: store, storage: storage}
}

func (s *Service) UpdateBio(ctx context.Context, user views.User, bio string) error {
	bio = strings.TrimSpace(bio)
	if len(bio) > 280 {
		return errors.New("bio must be 280 characters or fewer")
	}
	event := eventstore.DomainEvent{
		EventID:   uuid.NewString(),
		EventType: ProfileBioUpdated,
		Data: map[string]any{
			"profileBioUpdatedId": uuid.NewString(),
			"bio":                 bio,
			"updatedAt":           time.Now().Format(time.RFC3339),
			"scope":               map[string]any{"userRegisteredId": user.UserRegisteredID},
		},
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{}); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO profile_stats (user_id, name, username, email, bio, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4, $5, 0, 0)
		ON CONFLICT (user_id) DO UPDATE SET bio = EXCLUDED.bio, updated_at = now()
	`, user.UserRegisteredID, user.Name, user.Username, user.Email, bio)
	return err
}

func (s *Service) UploadImage(ctx context.Context, user views.User, data []byte, contentType string, header bool) (string, error) {
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
	event := eventstore.DomainEvent{
		EventID:   uuid.NewString(),
		EventType: eventType,
		Data: map[string]any{
			"imageUrl":   url,
			"uploadedAt": time.Now().Format(time.RFC3339),
			"scope":      map[string]any{"userRegisteredId": user.UserRegisteredID},
		},
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{}); err != nil {
		return "", err
	}
	column := "image"
	if header {
		column = "header_image_url"
		_, err := s.db.Exec(ctx, `
			INSERT INTO profile_stats (user_id, name, username, email, header_image_url, last_event_commit_position, last_event_prepare_position)
			VALUES ($1, $2, $3, $4, $5, 0, 0)
			ON CONFLICT (user_id) DO UPDATE SET header_image_url = EXCLUDED.header_image_url, updated_at = now()
		`, user.UserRegisteredID, user.Name, user.Username, user.Email, url)
		return url, err
	}
	_, err := s.db.Exec(ctx, `UPDATE auth_user SET image = $1, updated_at = now() WHERE id = $2`, url, user.ID)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(ctx, fmt.Sprintf(`
		INSERT INTO profile_stats (user_id, name, username, email, %s, last_event_commit_position, last_event_prepare_position)
		VALUES ($1, $2, $3, $4, $5, 0, 0)
		ON CONFLICT (user_id) DO UPDATE SET %s = EXCLUDED.%s, updated_at = now()
	`, column, column, column), user.UserRegisteredID, user.Name, user.Username, user.Email, url)
	return url, err
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
