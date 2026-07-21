package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/protectedpii"
)

const AuthUserProjectionEventHandlerName = "auth_user_projection_event_handler"

type AuthUserProjectionWriter interface {
	CreateRegisteredUserAccount(ctx context.Context, registered RegisterUserResult) error
	MarkEmailVerified(ctx context.Context, userRegisteredID string) error
	UpdateNameByRegisteredID(ctx context.Context, userRegisteredID, name string) error
	UpdatePasswordByRegisteredID(ctx context.Context, userRegisteredID, passwordHash string) error
}

type AuthVerificationProjectionWriter interface {
	CreateEmailVerificationOTP(ctx context.Context, userRegisteredID, otpID, code string, expiresAt time.Time) error
	CreatePasswordReset(ctx context.Context, userRegisteredID, requestID, token string, expiresAt time.Time) error
}

type AuthUserProjectionEventHandler struct {
	global        *eventstore.GlobalEventHandler
	writer        AuthUserProjectionWriter
	verifications AuthVerificationProjectionWriter
	retriever     eventstore.Retriever
	keys          SubjectPiiKeyPort
}

func NewAuthUserProjectionEventHandler(
	subscriber eventstore.Subscriber,
	checkpointer eventstore.Checkpointer,
	retriever eventstore.Retriever,
	writer AuthUserProjectionWriter,
	verifications AuthVerificationProjectionWriter,
	keys SubjectPiiKeyPort,
	logger *slog.Logger) (*AuthUserProjectionEventHandler, error) {
	handler := &AuthUserProjectionEventHandler{retriever: retriever, writer: writer, verifications: verifications, keys: keys}
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
	switch resolved.Event.EventType {
	case UserRegistered:
		return h.handleUserRegistered(ctx, resolved)
	case UserNameChanged:
		return h.handleUserNameChanged(ctx, resolved)
	case EmailVerificationOTPGenerated:
		return h.handleEmailVerificationOTPGenerated(ctx, resolved)
	case EmailVerificationOTPValidated:
		return h.handleEmailVerificationOTPValidated(ctx, resolved)
	case PasswordResetRequested:
		return h.handlePasswordResetRequested(ctx, resolved)
	case PasswordResetCompleted, PasswordChanged:
		return h.handlePasswordUpdated(ctx, resolved)
	default:
		return nil
	}
}

func (h *AuthUserProjectionEventHandler) handleUserRegistered(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	protector := protectedpii.FromEnv()
	userRegisteredID := stringValue(resolved.Event.Data[UserRegisteredIDField])
	subjectKey, ok, err := h.keys.GetSubjectDataKey(ctx, userRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	firstName := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, resolved.Event.Data, UserRegisteredFirstNameField)
	lastName := protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, resolved.Event.Data, UserRegisteredLastNameField)
	return h.writer.CreateRegisteredUserAccount(ctx, RegisterUserResult{
		UserRegisteredID: userRegisteredID,
		Username:         protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, resolved.Event.Data, UserRegisteredUsernameField),
		Email:            protectedpii.MustDecryptEventStringWithDataKey(protector, subjectKey, resolved.Event.Data, UserRegisteredEmailField),
		FirstName:        firstName,
		LastName:         lastName,
		PasswordHash:     stringValue(resolved.Event.Data[UserRegisteredPasswordHashField]),
	})
}

func (h *AuthUserProjectionEventHandler) handleUserNameChanged(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID := stringValue(eventstore.Scope(resolved.Event.Data)["userRegisteredId"])
	subjectKey, ok, err := h.keys.GetSubjectDataKey(ctx, userRegisteredID)
	if err != nil {
		return err
	}
	if !ok {
		return eventstore.ErrNotFound
	}
	name := protectedpii.MustDecryptEventStringWithDataKey(protectedpii.FromEnv(), subjectKey, resolved.Event.Data, UserNameChangedNameField)
	if userRegisteredID == "" || strings.TrimSpace(name) == "" {
		return nil
	}
	return h.writer.UpdateNameByRegisteredID(ctx, userRegisteredID, name)
}

func (h *AuthUserProjectionEventHandler) handleEmailVerificationOTPGenerated(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID := stringValue(eventstore.Scope(resolved.Event.Data)["userRegisteredId"])
	otpID := stringValue(resolved.Event.Data[EmailVerificationOTPGeneratedIDField])
	code := stringValue(resolved.Event.Data[EmailVerificationOTPCodeField])
	expiresAt, err := time.Parse(time.RFC3339, stringValue(resolved.Event.Data[EmailVerificationOTPExpiresAtField]))
	if userRegisteredID == "" || otpID == "" || code == "" || err != nil {
		return nil
	}
	return h.verifications.CreateEmailVerificationOTP(ctx, userRegisteredID, otpID, code, expiresAt)
}

func (h *AuthUserProjectionEventHandler) handleEmailVerificationOTPValidated(ctx context.Context, resolved eventstore.ResolvedEvent) error {
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

func (h *AuthUserProjectionEventHandler) handlePasswordResetRequested(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID := stringValue(eventstore.Scope(resolved.Event.Data)["userRegisteredId"])
	requestID := stringValue(resolved.Event.Data[PasswordResetRequestedIDField])
	token := stringValue(resolved.Event.Data[PasswordResetRequestedTokenField])
	expiresAt, err := time.Parse(time.RFC3339, stringValue(resolved.Event.Data[PasswordResetRequestedExpiresAtField]))
	if userRegisteredID == "" || requestID == "" || token == "" || err != nil {
		return nil
	}
	return h.verifications.CreatePasswordReset(ctx, userRegisteredID, requestID, token, expiresAt)
}

func (h *AuthUserProjectionEventHandler) handlePasswordUpdated(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	userRegisteredID := stringValue(eventstore.Scope(resolved.Event.Data)["userRegisteredId"])
	passwordHash := stringValue(resolved.Event.Data[PasswordChangedPasswordHashField])
	if resolved.Event.EventType == PasswordResetCompleted {
		passwordHash = stringValue(resolved.Event.Data[PasswordResetCompletedPasswordHashField])
	}
	if userRegisteredID == "" || passwordHash == "" {
		return nil
	}
	return h.writer.UpdatePasswordByRegisteredID(ctx, userRegisteredID, passwordHash)
}

func authUserProjectionEventHandlerQuery() eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserNameChanged}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPGenerated}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: PasswordResetRequested}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: PasswordResetCompleted}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: PasswordChanged}}},
	}}
}

func stringValue(value any) string {
	v, _ := value.(string)
	return v
}
