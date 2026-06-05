package profile

import (
	"context"
	"log/slog"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type ReadModelEventHandler struct {
	global    *eventstore.GlobalEventHandler
	readModel *ReadModel
}

func NewReadModelEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, readModel *ReadModel, logger *slog.Logger) (*ReadModelEventHandler, error) {
	handler := &ReadModelEventHandler{readModel: readModel}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            "profile_read_model_event_handler",
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
	switch resolved.Event.EventType {
	case userRegistered:
		return h.readModel.UpsertRegisteredUser(ctx, resolved)
	case userNameChanged:
		return h.readModel.UpdateName(ctx, resolved)
	case ProfileBioUpdated:
		return h.readModel.UpdateBio(ctx, resolved)
	case ProfileImageUploaded:
		return h.readModel.UpdateImage(ctx, resolved)
	case ProfileHeaderImageUploaded:
		return h.readModel.UpdateHeaderImage(ctx, resolved)
	default:
		return nil
	}
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
