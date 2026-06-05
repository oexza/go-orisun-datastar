package eventcatalog

import (
	"github.com/example/hono-event-starter-go/internal/auth"
	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/features/profile"
	"github.com/example/hono-event-starter-go/internal/features/todo"
)

func BoundaryIndexes() []eventstore.BoundaryIndexDefinition {
	return []eventstore.BoundaryIndexDefinition{
		{
			Name:       "user_registered_email",
			Fields:     []string{auth.UserRegisteredEmailField},
			EventTypes: []string{auth.UserRegistered},
		},
		{
			Name:       "user_registered_username",
			Fields:     []string{auth.UserRegisteredUsernameField},
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
	}
}
