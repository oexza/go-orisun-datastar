# Orisun Event Store Indexes

Indexes are declared in `internal/eventcatalog/indexes.go` and ensured at startup.

PII lookup indexes use protected blind-index fields, not plaintext event fields. See `docs/protected-pii-events.md`.

| Index | Fields | Event Types | Query Shape |
| --- | --- | --- | --- |
| `user_registered_email` | `emailHash` | `UserRegistered` | Registration duplicate email check. |
| `user_registered_username` | `usernameHash` | `UserRegistered` | Registration duplicate username check. |
| `user_registered_id` | `userRegisteredId` | `UserRegistered` | User existence/context loads. |
| `email_otp_generated_id` | `emailVerificationOTPGeneratedId` | `EmailVerificationOTPGenerated` | OTP send/validation context lookup. |
| `scope_email_otp_generated_id` | `scope.emailVerificationOTPGeneratedId` | `EmailVerificationOTPValidated`, `EmailVerificationOTPSent` | OTP sent/validated idempotency. |
| `password_reset_requested_id` | `passwordResetRequestedId` | `PasswordResetRequested` | Password reset email/send completion context. |
| `scope_password_reset_requested_id` | `scope.passwordResetRequestedId` | `PasswordResetEmailSent`, `PasswordResetCompleted` | Password reset idempotency and completion. |
| `scope_user_registered_id` | `scope.userRegisteredId` | Auth/profile/todo/account events | Per-user stream reconstruction and projections. |
| `todo_scope` | `scope.todoId`, `scope.userRegisteredId` | Todo lifecycle events | Per-todo ownership and idempotency context. |
| `account_deletion_requested_id` | `accountDeletionRequestedId` | `AccountDeletionRequested` | Account deletion completion lookup. |
| `scope_account_deletion_requested_id` | `scope.accountDeletionRequestedId` | `AccountDeleted` | Account deletion idempotency. |
| `login_attempt_recorded_id` | `loginAttemptRecordedId` | `LoginAttemptRecorded` | Login-attempt append subset/idempotency. |
| `login_attempt_identifier_hash` | `attemptedIdentifierHash` | `LoginAttemptRecorded` | Security audit lookup by attempted email/username blind index. |
| `login_attempt_ip_hash` | `ipAddressHash` | `LoginAttemptRecorded` | Security audit lookup by IP blind index. |
| `login_attempt_user_registered_id` | `userRegisteredId` | `LoginAttemptRecorded` | Successful login audit lookup by registered user. |

When adding an Orisun query or save subset query, add an explicit `eventType` criterion and update this table with the matching index.
