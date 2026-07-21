package profile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/OrisunLabs/go-orisun-datastar/internal/auth"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
)

const ProfileReadModelEventHandlerName = "profile_read_model_event_handler"

type ReadModelEventHandler struct {
	global    *eventstore.GlobalEventHandler
	readModel *ReadModel
	publisher eventstore.Publisher
	keys      auth.SubjectPiiKeyPort
}

func NewReadModelEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, readModel *ReadModel, publisher eventstore.Publisher, keys auth.SubjectPiiKeyPort, logger *slog.Logger) (*ReadModelEventHandler, error) {
	handler := &ReadModelEventHandler{readModel: readModel, publisher: publisher, keys: keys}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            ProfileReadModelEventHandlerName,
		Query:           readModelEventHandlerQuery(),
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

func (h *ReadModelEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *ReadModelEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *ReadModelEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	var userRegisteredID string
	switch resolved.Event.EventType {
	case userRegistered:
		userRegisteredID, _ = resolved.Event.Data["userRegisteredId"].(string)
		if err := h.readModel.UpsertRegisteredUser(ctx, resolved, h.keys); err != nil {
			return err
		}
	case userNameChanged:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateName(ctx, resolved, h.keys); err != nil {
			return err
		}
	case ProfileBioUpdated:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateBio(ctx, resolved, h.keys); err != nil {
			return err
		}
	case ProfileImageUploaded:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateImage(ctx, resolved); err != nil {
			return err
		}
	case ProfileHeaderImageUploaded:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateHeaderImage(ctx, resolved); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unhandled profile read model event type %q", resolved.Event.EventType)
	}
	if userRegisteredID == "" {
		return nil
	}
	return h.publisher.Publish(ctx, Channel(userRegisteredID), map[string]string{"userRegisteredId": userRegisteredID})
}

func readModelEventHandlerQuery() eventstore.Query {
	eventTypes := []string{
		userRegistered,
		userNameChanged,
		ProfileBioUpdated,
		ProfileImageUploaded,
		ProfileHeaderImageUploaded,
	}
	criteria := make([]eventstore.Criterion, 0, len(eventTypes))
	for _, eventType := range eventTypes {
		criteria = append(criteria, eventstore.Criterion{Tags: []eventstore.Tag{{Key: "eventType", Value: eventType}}})
	}
	return eventstore.Query{Criteria: criteria}
}
