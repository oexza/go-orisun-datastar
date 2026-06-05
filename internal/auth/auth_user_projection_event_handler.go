package auth

import (
	"context"
	"log/slog"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const AuthUserProjectionEventHandlerName = "auth_user_projection_event_handler"

type AuthUserProjectionWriter interface {
	MarkEmailVerified(ctx context.Context, userRegisteredID string) error
}

type AuthUserProjectionEventHandler struct {
	global    *eventstore.GlobalEventHandler
	writer    AuthUserProjectionWriter
	retriever eventstore.Retriever
}

func NewAuthUserProjectionEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, retriever eventstore.Retriever, writer AuthUserProjectionWriter, logger *slog.Logger) (*AuthUserProjectionEventHandler, error) {
	handler := &AuthUserProjectionEventHandler{retriever: retriever, writer: writer}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            AuthUserProjectionEventHandlerName,
		Query:           authUserProjectionEventHandlerQuery(),
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

func (h *AuthUserProjectionEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *AuthUserProjectionEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *AuthUserProjectionEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	if resolved.Event.EventType != EmailVerificationOTPValidated {
		return nil
	}
	userRegisteredID, _ := eventstore.Scope(resolved.Event.Data)["userRegisteredId"].(string)
	if userRegisteredID == "" {
		otpID, _ := eventstore.Scope(resolved.Event.Data)["emailVerificationOTPGeneratedId"].(string)
		generatedEvents, err := h.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, emailVerificationOTPGeneratedQuery(otpID))
		if err != nil {
			return err
		}
		if len(generatedEvents) == 0 {
			return nil
		}
		userRegisteredID, _ = eventstore.Scope(generatedEvents[0].Event.Data)["userRegisteredId"].(string)
	}
	if userRegisteredID == "" {
		return nil
	}
	return h.writer.MarkEmailVerified(ctx, userRegisteredID)
}

func authUserProjectionEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}}},
	}}
}
