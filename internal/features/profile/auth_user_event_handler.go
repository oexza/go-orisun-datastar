package profile

import (
	"context"
	"log/slog"

	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
)

const ProfileImageUploadedAuthUserEventHandlerName = "profile_image_uploaded_better_auth_event_handler"

type AuthUserImageBridge interface {
	UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error
}

type ProfileImageUploadedAuthUserEventHandler struct {
	global    *eventstore.GlobalEventHandler
	bridge    AuthUserImageBridge
	publisher eventstore.Publisher
}

func NewProfileImageUploadedAuthUserEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, bridge AuthUserImageBridge, publisher eventstore.Publisher, logger *slog.Logger) (*ProfileImageUploadedAuthUserEventHandler, error) {
	handler := &ProfileImageUploadedAuthUserEventHandler{bridge: bridge, publisher: publisher}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            ProfileImageUploadedAuthUserEventHandlerName,
		Query:           profileImageUploadedAuthUserEventHandlerQuery(),
		Logger:          logger,
		MaxEventRetries: -1,
		HandleEvent:     handler.handle,
	})
	if err != nil {
		return nil, err
	}
	handler.global = global
	return handler, nil
}

func (h *ProfileImageUploadedAuthUserEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *ProfileImageUploadedAuthUserEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *ProfileImageUploadedAuthUserEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	if resolved.Event.EventType != ProfileImageUploaded {
		return nil
	}
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	imageURL, _ := resolved.Event.Data["imageUrl"].(string)
	if userRegisteredID == "" || imageURL == "" {
		return nil
	}
	if err := h.bridge.UpdateImage(ctx, userRegisteredID, imageURL); err != nil {
		return err
	}
	return h.publisher.Publish(ctx, Channel(userRegisteredID), map[string]string{"userRegisteredId": userRegisteredID})
}

func profileImageUploadedAuthUserEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileImageUploaded}}}}}
}
