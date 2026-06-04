package profile

import (
	"context"
	"log/slog"

	"github.com/example/hono-event-starter-go/internal/eventstore"
)

const ProfileImageUploadedAuthUserEventHandlerName = "profile_image_uploaded_better_auth_event_handler"

type AuthUserImageBridge interface {
	UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error
}

type ProfileImageUploadedAuthUserEventHandler struct {
	global *eventstore.GlobalEventHandler
	bridge AuthUserImageBridge
}

func NewProfileImageUploadedAuthUserEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, bridge AuthUserImageBridge, logger *slog.Logger) (*ProfileImageUploadedAuthUserEventHandler, error) {
	handler := &ProfileImageUploadedAuthUserEventHandler{bridge: bridge}
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
	return h.bridge.UpdateImage(ctx, userRegisteredID, imageURL)
}

func profileImageUploadedAuthUserEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileImageUploaded}}}}}
}
