package auth

import (
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

const (
	UserRegistered                = "UserRegistered"
	UserNameChanged               = "UserNameChanged"
	EmailVerificationOTPGenerated = "EmailVerificationOTPGenerated"
	EmailVerificationOTPValidated = "EmailVerificationOTPValidated"
	EmailVerificationOTPSent      = "EmailVerificationOTPSent"
	PasswordResetRequested        = "PasswordResetRequested"
	PasswordResetEmailSent        = "PasswordResetEmailSent"
	PasswordResetCompleted        = "PasswordResetCompleted"
	PasswordChanged               = "PasswordChanged"
)

const (
	UserRegisteredIDField                     = "userRegisteredId"
	UserRegisteredUsernameField               = "username"
	UserRegisteredEmailField                  = "email"
	UserRegisteredFirstNameField              = "firstName"
	UserRegisteredLastNameField               = "lastName"
	UserRegisteredYearOfBirthField            = "yearOfBirth"
	UserRegisteredPasswordHashField           = "passwordHash"
	UserNameChangedIDField                    = "userNameChangedId"
	UserNameChangedNameField                  = "name"
	UserNameChangedChangedAtField             = "changedAt"
	EmailVerificationOTPGeneratedIDField      = "emailVerificationOTPGeneratedId"
	EmailVerificationOTPCodeField             = "otpCode"
	EmailVerificationOTPExpiresAtField        = "expiresAt"
	EmailVerificationOTPValidatedIDField      = "emailVerificationOTPValidatedId"
	EmailVerificationOTPValidatedAtField      = "validatedAt"
	EmailVerificationOTPSentIDField           = "emailVerificationOTPSentId"
	EmailVerificationOTPSentAtField           = "sentAt"
	PasswordResetRequestedIDField             = "passwordResetRequestedId"
	PasswordResetRequestedEmailField          = "email"
	PasswordResetRequestedTokenField          = "resetToken"
	PasswordResetRequestedExpiresAtField      = "expiresAt"
	PasswordResetEmailSentIDField             = "passwordResetEmailSentId"
	PasswordResetEmailSentAtField             = "sentAt"
	PasswordResetCompletedIDField             = "passwordResetCompletedId"
	PasswordResetCompletedResetAtField        = "resetAt"
	PasswordResetCompletedPasswordHashField   = "passwordHash"
	PasswordChangedIDField                    = "passwordChangedId"
	PasswordChangedAtField                    = "changedAt"
	PasswordChangedPasswordHashField          = "passwordHash"
	ScopeUserRegisteredIDField                = "scope.userRegisteredId"
	ScopeEmailVerificationOTPGeneratedIDField = "scope.emailVerificationOTPGeneratedId"
	ScopePasswordResetRequestedIDField        = "scope.passwordResetRequestedId"
)

type UserRegisteredEvent struct {
	UserRegisteredID string         `json:"userRegisteredId"`
	Username         string         `json:"username"`
	Email            string         `json:"email"`
	FirstName        string         `json:"firstName"`
	LastName         string         `json:"lastName"`
	YearOfBirth      int            `json:"yearOfBirth"`
	PasswordHash     string         `json:"passwordHash"`
	Scope            map[string]any `json:"scope"`
}

type UserNameChangedEvent struct {
	UserNameChangedID string              `json:"userNameChangedId"`
	Name              string              `json:"name"`
	ChangedAt         string              `json:"changedAt"`
	Scope             UserRegisteredScope `json:"scope"`
}

type EmailVerificationOTPGeneratedEvent struct {
	EmailVerificationOTPGeneratedID string              `json:"emailVerificationOTPGeneratedId"`
	OTPCode                         string              `json:"otpCode"`
	ExpiresAt                       string              `json:"expiresAt"`
	Scope                           UserRegisteredScope `json:"scope"`
}

type EmailVerificationOTPValidatedEvent struct {
	EmailVerificationOTPValidatedID string                             `json:"emailVerificationOTPValidatedId"`
	ValidatedAt                     string                             `json:"validatedAt"`
	Scope                           EmailVerificationOTPValidatedScope `json:"scope"`
}

type EmailVerificationOTPSentEvent struct {
	EmailVerificationOTPSentID string                             `json:"emailVerificationOTPSentId"`
	SentAt                     string                             `json:"sentAt"`
	Scope                      EmailVerificationOTPGeneratedScope `json:"scope"`
}

type PasswordResetRequestedEvent struct {
	PasswordResetRequestedID string              `json:"passwordResetRequestedId"`
	Email                    string              `json:"email"`
	ResetToken               string              `json:"resetToken"`
	ExpiresAt                string              `json:"expiresAt"`
	Scope                    UserRegisteredScope `json:"scope"`
}

type PasswordResetEmailSentEvent struct {
	PasswordResetEmailSentID string                      `json:"passwordResetEmailSentId"`
	SentAt                   string                      `json:"sentAt"`
	Scope                    PasswordResetRequestedScope `json:"scope"`
}

type PasswordResetCompletedEvent struct {
	PasswordResetCompletedID string                      `json:"passwordResetCompletedId"`
	ResetAt                  string                      `json:"resetAt"`
	PasswordHash             string                      `json:"passwordHash"`
	Scope                    PasswordResetCompletedScope `json:"scope"`
}

type PasswordChangedEvent struct {
	PasswordChangedID string              `json:"passwordChangedId"`
	ChangedAt         string              `json:"changedAt"`
	PasswordHash      string              `json:"passwordHash"`
	Scope             UserRegisteredScope `json:"scope"`
}

type UserRegisteredScope struct {
	UserRegisteredID string `json:"userRegisteredId"`
}

type EmailVerificationOTPGeneratedScope struct {
	EmailVerificationOTPGeneratedID string `json:"emailVerificationOTPGeneratedId"`
}

type EmailVerificationOTPValidatedScope struct {
	EmailVerificationOTPGeneratedID string `json:"emailVerificationOTPGeneratedId"`
	UserRegisteredID                string `json:"userRegisteredId"`
}

type PasswordResetRequestedScope struct {
	PasswordResetRequestedID string `json:"passwordResetRequestedId"`
}

type PasswordResetCompletedScope struct {
	PasswordResetRequestedID string `json:"passwordResetRequestedId"`
	UserRegisteredID         string `json:"userRegisteredId"`
}

func NewUserRegisteredEvent(userRegisteredID, username, emailAddress, firstName, lastName string, yearOfBirth int, passwordHash string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   userRegisteredID,
		EventType: UserRegistered,
		Data: eventstore.MustData(UserRegisteredEvent{
			UserRegisteredID: userRegisteredID,
			Username:         username,
			Email:            emailAddress,
			FirstName:        firstName,
			LastName:         lastName,
			YearOfBirth:      yearOfBirth,
			PasswordHash:     passwordHash,
			Scope:            map[string]any{},
		}),
		Metadata: metadata,
	}
}

func NewUserNameChangedEvent(userNameChangedID, name string, changedAt time.Time, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   userNameChangedID,
		EventType: UserNameChanged,
		Data: eventstore.MustData(UserNameChangedEvent{
			UserNameChangedID: userNameChangedID,
			Name:              name,
			ChangedAt:         formatEventTime(changedAt),
			Scope:             UserRegisteredScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewEmailVerificationOTPGeneratedEvent(emailVerificationOTPGeneratedID, otpCode string, expiresAt time.Time, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   emailVerificationOTPGeneratedID,
		EventType: EmailVerificationOTPGenerated,
		Data: eventstore.MustData(EmailVerificationOTPGeneratedEvent{
			EmailVerificationOTPGeneratedID: emailVerificationOTPGeneratedID,
			OTPCode:                         otpCode,
			ExpiresAt:                       formatEventTime(expiresAt),
			Scope:                           UserRegisteredScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewEmailVerificationOTPValidatedEvent(emailVerificationOTPValidatedID string, validatedAt time.Time, emailVerificationOTPGeneratedID, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   emailVerificationOTPValidatedID,
		EventType: EmailVerificationOTPValidated,
		Data: eventstore.MustData(EmailVerificationOTPValidatedEvent{
			EmailVerificationOTPValidatedID: emailVerificationOTPValidatedID,
			ValidatedAt:                     formatEventTime(validatedAt),
			Scope: EmailVerificationOTPValidatedScope{
				EmailVerificationOTPGeneratedID: emailVerificationOTPGeneratedID,
				UserRegisteredID:                userRegisteredID,
			},
		}),
		Metadata: metadata,
	}
}

func NewEmailVerificationOTPSentEvent(emailVerificationOTPSentID string, sentAt time.Time, emailVerificationOTPGeneratedID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   emailVerificationOTPSentID,
		EventType: EmailVerificationOTPSent,
		Data: eventstore.MustData(EmailVerificationOTPSentEvent{
			EmailVerificationOTPSentID: emailVerificationOTPSentID,
			SentAt:                     formatEventTime(sentAt),
			Scope:                      EmailVerificationOTPGeneratedScope{EmailVerificationOTPGeneratedID: emailVerificationOTPGeneratedID},
		}),
		Metadata: metadata,
	}
}

func NewPasswordResetRequestedEvent(passwordResetRequestedID, emailAddress, resetToken string, expiresAt time.Time, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   passwordResetRequestedID,
		EventType: PasswordResetRequested,
		Data: eventstore.MustData(PasswordResetRequestedEvent{
			PasswordResetRequestedID: passwordResetRequestedID,
			Email:                    emailAddress,
			ResetToken:               resetToken,
			ExpiresAt:                formatEventTime(expiresAt),
			Scope:                    UserRegisteredScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewPasswordResetEmailSentEvent(passwordResetEmailSentID string, sentAt time.Time, passwordResetRequestedID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   passwordResetEmailSentID,
		EventType: PasswordResetEmailSent,
		Data: eventstore.MustData(PasswordResetEmailSentEvent{
			PasswordResetEmailSentID: passwordResetEmailSentID,
			SentAt:                   formatEventTime(sentAt),
			Scope:                    PasswordResetRequestedScope{PasswordResetRequestedID: passwordResetRequestedID},
		}),
		Metadata: metadata,
	}
}

func NewPasswordResetCompletedEvent(passwordResetCompletedID string, resetAt time.Time, passwordResetRequestedID, userRegisteredID, passwordHash string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   passwordResetCompletedID,
		EventType: PasswordResetCompleted,
		Data: eventstore.MustData(PasswordResetCompletedEvent{
			PasswordResetCompletedID: passwordResetCompletedID,
			ResetAt:                  formatEventTime(resetAt),
			PasswordHash:             passwordHash,
			Scope: PasswordResetCompletedScope{
				PasswordResetRequestedID: passwordResetRequestedID,
				UserRegisteredID:         userRegisteredID,
			},
		}),
		Metadata: metadata,
	}
}

func NewPasswordChangedEvent(passwordChangedID string, changedAt time.Time, userRegisteredID, passwordHash string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   passwordChangedID,
		EventType: PasswordChanged,
		Data: eventstore.MustData(PasswordChangedEvent{
			PasswordChangedID: passwordChangedID,
			ChangedAt:         formatEventTime(changedAt),
			PasswordHash:      passwordHash,
			Scope:             UserRegisteredScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func formatEventTime(value time.Time) string {
	return value.Format(time.RFC3339)
}
