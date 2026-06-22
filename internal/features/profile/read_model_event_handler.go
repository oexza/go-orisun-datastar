package profile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const ProfileReadModelEventHandlerName = "profile_read_model_event_handler"

type ReadModelEventHandler struct {
	global    *eventstore.GlobalEventHandler
	readModel *ReadModel
	publisher eventstore.Publisher
}

func NewReadModelEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, readModel *ReadModel, publisher eventstore.Publisher, logger *slog.Logger) (*ReadModelEventHandler, error) {
	handler := &ReadModelEventHandler{readModel: readModel, publisher: publisher}
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
		if err := h.readModel.UpsertRegisteredUser(ctx, resolved); err != nil {
			return err
		}
	case userNameChanged:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateName(ctx, resolved); err != nil {
			return err
		}
	case ProfileBioUpdated:
		userRegisteredID, _ = eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
		if err := h.readModel.UpdateBio(ctx, resolved); err != nil {
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
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: userRegistered}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: userNameChanged}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileBioUpdated}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileImageUploaded}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: ProfileHeaderImageUploaded}}},
	}}
}
