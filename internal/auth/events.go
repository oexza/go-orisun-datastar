package auth

import (
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/protectedpii"
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
	UserRegisteredUsernameHashField           = "usernameHash"
	UserRegisteredEmailField                  = "email"
	UserRegisteredEmailHashField              = "emailHash"
	UserRegisteredFirstNameField              = "firstName"
	UserRegisteredLastNameField               = "lastName"
	UserRegisteredYearOfBirthField            = "yearOfBirth"
	UserRegisteredPasswordHashField           = "passwordHash"
	UserNameChangedIDField                    = "userNameChangedId"
	UserNameChangedNameField                  = "name"
	UserNameChangedNameHashField              = "nameHash"
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
	PasswordResetRequestedEmailHashField      = "emailHash"
	PasswordResetRequestedTokenField          = "resetToken"
	PasswordResetRequestedTokenHashField      = "resetTokenHash"
	PasswordResetRequestedExpiresAtField      = "expiresAt"
	PasswordResetEmailSentIDField             = "passwordResetEmailSentId"
	PasswordResetEmailSentAtField             = "sentAt"
	PasswordResetCompletedIDField             = "passwordResetCompletedId"
	PasswordResetCompletedResetAtField        = "resetAt"
	PasswordResetCompletedPasswordHashField   = "passwordHash"
	PasswordChangedIDField                    = "passwordChangedId"
	PasswordChangedAtField                    = "changedAt"
	PasswordChangedPasswordHashField          = "passwordHash"
	ScopeUserRegisteredIDKey                  = UserRegisteredIDField
	ScopeEmailVerificationOTPGeneratedIDKey   = EmailVerificationOTPGeneratedIDField
	ScopePasswordResetRequestedIDKey          = PasswordResetRequestedIDField
	ScopeUserRegisteredIDField                = "scope." + ScopeUserRegisteredIDKey
	ScopeEmailVerificationOTPGeneratedIDField = "scope." + ScopeEmailVerificationOTPGeneratedIDKey
	ScopePasswordResetRequestedIDField        = "scope." + ScopePasswordResetRequestedIDKey
	AccountDeletionRequested                  = "AccountDeletionRequested"
	AccountDeleted                            = "AccountDeleted"
	AccountDeletionRequestedIDField           = "accountDeletionRequestedId"
	AccountDeletionAuthUserIDField            = "authUserId"
	AccountDeletedIDField                     = "accountDeletedId"
	AccountDeletionRequestedAtField           = "requestedAt"
	AccountDeletedAtField                     = "deletedAt"
	LoginAttemptRecorded                      = "LoginAttemptRecorded"
	LoginAttemptRecordedIDField               = "loginAttemptRecordedId"
	LoginAttemptIdentifierHashField           = "attemptedIdentifierHash"
	LoginAttemptIPAddressHashField            = "ipAddressHash"
	LoginAttemptUserRegisteredIDField         = "userRegisteredId"
)

type UserRegisteredEvent struct {
	UserRegisteredID string             `json:"userRegisteredId"`
	Username         protectedpii.Value `json:"username"`
	UsernameHash     string             `json:"usernameHash"`
	Email            protectedpii.Value `json:"email"`
	EmailHash        string             `json:"emailHash"`
	FirstName        protectedpii.Value `json:"firstName"`
	LastName         protectedpii.Value `json:"lastName"`
	YearOfBirth      int                `json:"yearOfBirth"`
	PasswordHash     string             `json:"passwordHash"`
	Scope            map[string]any     `json:"scope"`
}

type UserNameChangedEvent struct {
	UserNameChangedID string              `json:"userNameChangedId"`
	Name              protectedpii.Value  `json:"name"`
	NameHash          string              `json:"nameHash"`
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
	Email                    protectedpii.Value  `json:"email"`
	EmailHash                string              `json:"emailHash"`
	ResetToken               protectedpii.Value  `json:"resetToken"`
	ResetTokenHash           string              `json:"resetTokenHash"`
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

type AccountDeletionRequestedEvent struct {
	AccountDeletionRequestedID string              `json:"accountDeletionRequestedId"`
	RequestedAt                string              `json:"requestedAt"`
	AuthUserID                 string              `json:"authUserId"`
	Scope                      UserRegisteredScope `json:"scope"`
}

type AccountDeletedEvent struct {
	AccountDeletedID           string               `json:"accountDeletedId"`
	DeletedAt                  string               `json:"deletedAt"`
	AccountDeletionRequestedID string               `json:"accountDeletionRequestedId"`
	AuthUserID                 string               `json:"authUserId"`
	Scope                      AccountDeletionScope `json:"scope"`
}

type LoginAttemptRecordedEvent struct {
	LoginAttemptRecordedID  string             `json:"loginAttemptRecordedId"`
	AttemptedIdentifier     protectedpii.Value `json:"attemptedIdentifier"`
	AttemptedIdentifierHash string             `json:"attemptedIdentifierHash"`
	IPAddress               protectedpii.Value `json:"ipAddress"`
	IPAddressHash           string             `json:"ipAddressHash"`
	UserRegisteredID        string             `json:"userRegisteredId,omitempty"`
	Succeeded               bool               `json:"succeeded"`
	RecordedAt              string             `json:"recordedAt"`
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

type AccountDeletionScope struct {
	AccountDeletionRequestedID string `json:"accountDeletionRequestedId"`
	UserRegisteredID           string `json:"userRegisteredId"`
}

func NewUserRegisteredEvent(userRegisteredID, username, emailAddress, firstName, lastName string, yearOfBirth int, passwordHash string, subjectKey protectedpii.SubjectDataKey, metadata map[string]any) eventstore.DomainEvent {
	protector := protectedpii.FromEnv()
	return eventstore.DomainEvent{
		EventID:   userRegisteredID,
		EventType: UserRegistered,
		Data: eventstore.MustData(UserRegisteredEvent{
			UserRegisteredID: userRegisteredID,
			Username:         protector.MustProtectWithDataKey(username, UserRegisteredUsernameField, subjectKey),
			UsernameHash:     protector.BlindIndex(UserRegisteredUsernameField, username),
			Email:            protector.MustProtectWithDataKey(emailAddress, UserRegisteredEmailField, subjectKey),
			EmailHash:        protector.BlindIndex(UserRegisteredEmailField, emailAddress),
			FirstName:        protector.MustProtectWithDataKey(firstName, UserRegisteredFirstNameField, subjectKey),
			LastName:         protector.MustProtectWithDataKey(lastName, UserRegisteredLastNameField, subjectKey),
			YearOfBirth:      yearOfBirth,
			PasswordHash:     passwordHash,
			Scope:            map[string]any{},
		}),
		Metadata: metadata,
	}
}

func NewUserNameChangedEvent(userNameChangedID, name string, changedAt time.Time, userRegisteredID string, subjectKey protectedpii.SubjectDataKey, metadata map[string]any) eventstore.DomainEvent {
	protector := protectedpii.FromEnv()
	return eventstore.DomainEvent{
		EventID:   userNameChangedID,
		EventType: UserNameChanged,
		Data: eventstore.MustData(UserNameChangedEvent{
			UserNameChangedID: userNameChangedID,
			Name:              protector.MustProtectWithDataKey(name, UserNameChangedNameField, subjectKey),
			NameHash:          protector.BlindIndex(UserNameChangedNameField, name),
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

func NewPasswordResetRequestedEvent(passwordResetRequestedID, emailAddress, resetToken string, expiresAt time.Time, userRegisteredID string, subjectKey protectedpii.SubjectDataKey, metadata map[string]any) eventstore.DomainEvent {
	protector := protectedpii.FromEnv()
	return eventstore.DomainEvent{
		EventID:   passwordResetRequestedID,
		EventType: PasswordResetRequested,
		Data: eventstore.MustData(PasswordResetRequestedEvent{
			PasswordResetRequestedID: passwordResetRequestedID,
			Email:                    protector.MustProtectWithDataKey(emailAddress, PasswordResetRequestedEmailField, subjectKey),
			EmailHash:                protector.BlindIndex(PasswordResetRequestedEmailField, emailAddress),
			ResetToken:               protector.MustProtectWithDataKey(resetToken, PasswordResetRequestedTokenField, subjectKey),
			ResetTokenHash:           protector.SensitiveBlindIndex(PasswordResetRequestedTokenField, resetToken),
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

func NewAccountDeletionRequestedEvent(accountDeletionRequestedID string, requestedAt time.Time, authUserID, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   accountDeletionRequestedID,
		EventType: AccountDeletionRequested,
		Data: eventstore.MustData(AccountDeletionRequestedEvent{
			AccountDeletionRequestedID: accountDeletionRequestedID,
			RequestedAt:                formatEventTime(requestedAt),
			AuthUserID:                 authUserID,
			Scope:                      UserRegisteredScope{UserRegisteredID: userRegisteredID},
		}),
		Metadata: metadata,
	}
}

func NewAccountDeletedEvent(accountDeletedID string, deletedAt time.Time, accountDeletionRequestedID, authUserID, userRegisteredID string, metadata map[string]any) eventstore.DomainEvent {
	return eventstore.DomainEvent{
		EventID:   accountDeletedID,
		EventType: AccountDeleted,
		Data: eventstore.MustData(AccountDeletedEvent{
			AccountDeletedID:           accountDeletedID,
			DeletedAt:                  formatEventTime(deletedAt),
			AccountDeletionRequestedID: accountDeletionRequestedID,
			AuthUserID:                 authUserID,
			Scope: AccountDeletionScope{
				AccountDeletionRequestedID: accountDeletionRequestedID,
				UserRegisteredID:           userRegisteredID,
			},
		}),
		Metadata: metadata,
	}
}

func NewLoginAttemptRecordedEvent(loginAttemptRecordedID string, recordedAt time.Time, attemptedIdentifier, ipAddress, userRegisteredID string, succeeded bool, metadata map[string]any) eventstore.DomainEvent {
	protector := protectedpii.FromEnv()
	return eventstore.DomainEvent{
		EventID:   loginAttemptRecordedID,
		EventType: LoginAttemptRecorded,
		Data: eventstore.MustData(LoginAttemptRecordedEvent{
			LoginAttemptRecordedID:  loginAttemptRecordedID,
			AttemptedIdentifier:     protector.MustProtect(attemptedIdentifier),
			AttemptedIdentifierHash: protector.BlindIndex("attemptedIdentifier", attemptedIdentifier),
			IPAddress:               protector.MustProtect(ipAddress),
			IPAddressHash:           protector.SensitiveBlindIndex("ipAddress", ipAddress),
			UserRegisteredID:        userRegisteredID,
			Succeeded:               succeeded,
			RecordedAt:              formatEventTime(recordedAt),
		}),
		Metadata: metadata,
	}
}

func formatEventTime(value time.Time) string {
	return value.Format(time.RFC3339)
}
