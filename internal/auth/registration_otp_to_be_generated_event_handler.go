package auth

import (
	"context"
	"log/slog"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

const RegistrationOTPToBeGeneratedEventHandlerName = "todo_registration_otp_to_be_generated_event_handler"

type EmailVerificationOTPIssuer interface {
	GenerateEmailVerificationOTPWithMetadata(ctx context.Context, user views.User, metadata CommandMetadata) error
}

type RegistrationOTPToBeGeneratedEventHandler struct {
	global *eventstore.GlobalEventHandler
	issuer EmailVerificationOTPIssuer
}

func NewRegistrationOTPToBeGeneratedEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, issuer EmailVerificationOTPIssuer, logger *slog.Logger) (*RegistrationOTPToBeGeneratedEventHandler, error) {
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

func userRegisteredEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}}}}}
}
