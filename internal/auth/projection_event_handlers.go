package auth

import (
	"context"
	"log/slog"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

const (
	RegistrationOTPToBeGeneratedEventHandlerName = "todo_registration_otp_to_be_generated_event_handler"
	AuthUserProjectionEventHandlerName           = "auth_user_projection_event_handler"
)

type EmailVerificationOTPIssuer interface {
	GenerateEmailVerificationOTPWithMetadata(ctx context.Context, user views.User, metadata CommandMetadata) error
}

type AuthUserProjectionWriter interface {
	MarkEmailVerified(ctx context.Context, userRegisteredID string) error
}

type RegistrationOTPToBeGeneratedEventHandler struct {
	global *eventstore.GlobalEventHandler
	issuer EmailVerificationOTPIssuer
}

func NewRegistrationOTPToBeGeneratedEventHandler(subscriber EventSubscriber, checkpointer EventCheckpointer, issuer EmailVerificationOTPIssuer, logger *slog.Logger) (*RegistrationOTPToBeGeneratedEventHandler, error) {
	handler := &RegistrationOTPToBeGeneratedEventHandler{issuer: issuer}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            RegistrationOTPToBeGeneratedEventHandlerName,
		Query:           userRegisteredEventHandlerQuery(),
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

func (h *RegistrationOTPToBeGeneratedEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *RegistrationOTPToBeGeneratedEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *RegistrationOTPToBeGeneratedEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	if resolved.Event.EventType != UserRegistered {
		return nil
	}
	userRegisteredID, _ := resolved.Event.Data["userRegisteredId"].(string)
	emailAddress, _ := resolved.Event.Data["email"].(string)
	firstName, _ := resolved.Event.Data["firstName"].(string)
	lastName, _ := resolved.Event.Data["lastName"].(string)
	username, _ := resolved.Event.Data["username"].(string)
	return h.issuer.GenerateEmailVerificationOTPWithMetadata(ctx, views.User{
		UserRegisteredID: userRegisteredID,
		Name:             strings.TrimSpace(firstName + " " + lastName),
		Username:         username,
		Email:            emailAddress,
	}, eventstore.EventHandlerCommandMetadata(RegistrationOTPToBeGeneratedEventHandlerName, resolved))
}

type AuthUserProjectionEventHandler struct {
	global    *eventstore.GlobalEventHandler
	writer    AuthUserProjectionWriter
	retriever EventRetriever
}

func NewAuthUserProjectionEventHandler(subscriber EventSubscriber, checkpointer EventCheckpointer, retriever EventRetriever, writer AuthUserProjectionWriter, logger *slog.Logger) (*AuthUserProjectionEventHandler, error) {
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

func userRegisteredEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}}}}}
}

func authUserProjectionEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}}},
	}}
}
