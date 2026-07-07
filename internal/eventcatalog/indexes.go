package eventcatalog

import (
	"github.com/oexza/go-orisun-datastar/internal/auth"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/features/profile"
	"github.com/oexza/go-orisun-datastar/internal/features/todo"
)

func BoundaryIndexes() []eventstore.BoundaryIndexDefinition {
	return []eventstore.BoundaryIndexDefinition{
		{
			Name:       "user_registered_email",
			Fields:     []string{auth.UserRegisteredEmailHashField},
			EventTypes: []string{auth.UserRegistered},
		},
		{
			Name:       "user_registered_username",
			Fields:     []string{auth.UserRegisteredUsernameHashField},
			EventTypes: []string{auth.UserRegistered},
		},
		{
			Name:       "user_registered_id",
			Fields:     []string{auth.UserRegisteredIDField},
			EventTypes: []string{auth.UserRegistered},
		},
		{
			Name:       "email_otp_generated_id",
			Fields:     []string{auth.EmailVerificationOTPGeneratedIDField},
			EventTypes: []string{auth.EmailVerificationOTPGenerated},
		},
		{
			Name:       "scope_email_otp_generated_id",
			Fields:     []string{auth.ScopeEmailVerificationOTPGeneratedIDField},
			EventTypes: []string{auth.EmailVerificationOTPValidated, auth.EmailVerificationOTPSent},
		},
		{
			Name:       "password_reset_requested_id",
			Fields:     []string{auth.PasswordResetRequestedIDField},
			EventTypes: []string{auth.PasswordResetRequested},
		},
		{
			Name:       "scope_password_reset_requested_id",
			Fields:     []string{auth.ScopePasswordResetRequestedIDField},
			EventTypes: []string{auth.PasswordResetEmailSent, auth.PasswordResetCompleted},
		},
		{
			Name:   "scope_user_registered_id",
			Fields: []string{auth.ScopeUserRegisteredIDField},
			EventTypes: []string{
				auth.UserNameChanged,
				auth.EmailVerificationOTPGenerated,
				auth.EmailVerificationOTPValidated,
				auth.PasswordResetRequested,
				auth.PasswordResetCompleted,
				auth.PasswordChanged,
				auth.AccountDeletionRequested,
				auth.AccountDeleted,
				profile.ProfileImageUploaded,
				profile.ProfileHeaderImageUploaded,
				profile.ProfileBioUpdated,
				todo.TodoCreated,
				todo.TodoRenamed,
				todo.TodoCompleted,
				todo.TodoReopened,
				todo.TodoDeleted,
			},
		},
		{
			Name:       "todo_scope",
			Fields:     []string{todo.TodoScopeIDField, todo.TodoScopeUserRegisteredIDField},
			EventTypes: []string{todo.TodoCreated, todo.TodoRenamed, todo.TodoCompleted, todo.TodoReopened, todo.TodoDeleted},
		},
		{
			Name:       "account_deletion_requested_id",
			Fields:     []string{auth.AccountDeletionRequestedIDField},
			EventTypes: []string{auth.AccountDeletionRequested},
		},
		{
			Name:       "scope_account_deletion_requested_id",
			Fields:     []string{"scope.accountDeletionRequestedId"},
			EventTypes: []string{auth.AccountDeleted},
		},
		{
			Name:       "login_attempt_recorded_id",
			Fields:     []string{auth.LoginAttemptRecordedIDField},
			EventTypes: []string{auth.LoginAttemptRecorded},
		},
		{
			Name:       "login_attempt_identifier_hash",
			Fields:     []string{auth.LoginAttemptIdentifierHashField},
			EventTypes: []string{auth.LoginAttemptRecorded},
		},
		{
			Name:       "login_attempt_ip_hash",
			Fields:     []string{auth.LoginAttemptIPAddressHashField},
			EventTypes: []string{auth.LoginAttemptRecorded},
		},
		{
			Name:       "login_attempt_user_registered_id",
			Fields:     []string{auth.LoginAttemptUserRegisteredIDField},
			EventTypes: []string{auth.LoginAttemptRecorded},
		},
	}
}
