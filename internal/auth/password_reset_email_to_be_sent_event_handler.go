package auth

import (
	"context"
	"log/slog"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const PasswordResetEmailToBeSentEventHandlerName = "todo_password_reset_email_to_be_sent_event_handler"

type PasswordResetEmailToBeSentEventHandler struct {
	global    *eventstore.GlobalEventHandler
	retriever eventstore.Retriever
	saver     eventstore.Saver
	sender    EmailSender
	appURL    string
}

func NewPasswordResetEmailToBeSentEventHandler(subscriber eventstore.Subscriber, checkpointer eventstore.Checkpointer, retriever eventstore.Retriever, saver eventstore.Saver, sender EmailSender, appURL string, logger *slog.Logger) (*PasswordResetEmailToBeSentEventHandler, error) {
	handler := &PasswordResetEmailToBeSentEventHandler{retriever: retriever, saver: saver, sender: sender, appURL: appURL}
	global, err := eventstore.NewGlobalEventHandler(eventstore.GlobalEventHandlerConfig{
		Subscriber:      subscriber,
		Checkpointer:    checkpointer,
		Name:            PasswordResetEmailToBeSentEventHandlerName,
		Query:           passwordResetEmailToBeSentEventHandlerQuery(),
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

func (h *PasswordResetEmailToBeSentEventHandler) StartSubscribing(ctx context.Context) error {
	return h.global.StartSubscribing(ctx)
}

func (h *PasswordResetEmailToBeSentEventHandler) StopSubscribing() {
	h.global.StopSubscribing()
}

func (h *PasswordResetEmailToBeSentEventHandler) handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	if resolved.Event.EventType != PasswordResetRequested {
		return nil
	}
	requestID, _ := resolved.Event.Data["passwordResetRequestedId"].(string)
	return SendPasswordResetEmailCommandHandler(ctx, SendPasswordResetEmailCommand{
		PasswordResetRequestedID: requestID,
		Metadata:                 eventstore.EventHandlerCommandMetadata(PasswordResetEmailToBeSentEventHandlerName, resolved),
	}, h.saver, h.retriever, h.sender, h.appURL)
}

func passwordResetEmailToBeSentEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{{Key: "eventType", Value: PasswordResetRequested}}}}}
}
