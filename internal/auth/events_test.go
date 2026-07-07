package auth

import (
	"testing"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/protectedpii"
)

func TestPasswordResetCompletedEventMatchesFrasesShape(t *testing.T) {
	resetAt := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	event := NewPasswordResetCompletedEvent("completed-id", resetAt, "request-id", "user-id", "hash", nil)

	if event.EventID != "completed-id" {
		t.Fatalf("event id = %q, want completed-id", event.EventID)
	}
	if event.EventType != PasswordResetCompleted {
		t.Fatalf("event type = %q, want %q", event.EventType, PasswordResetCompleted)
	}
	if event.Data["passwordResetCompletedId"] != "completed-id" {
		t.Fatalf("passwordResetCompletedId = %v, want completed-id", event.Data["passwordResetCompletedId"])
	}
	if event.Data["resetAt"] != resetAt.Format(time.RFC3339) {
		t.Fatalf("resetAt = %v, want %s", event.Data["resetAt"], resetAt.Format(time.RFC3339))
	}
	if event.Data["passwordHash"] != "hash" {
		t.Fatalf("passwordHash = %v, want hash", event.Data["passwordHash"])
	}

	scope, ok := event.Data["scope"].(map[string]any)
	if !ok {
		t.Fatal("scope missing or wrong type")
	}
	if scope["passwordResetRequestedId"] != "request-id" {
		t.Fatalf("scope.passwordResetRequestedId = %v, want request-id", scope["passwordResetRequestedId"])
	}
	if scope["userRegisteredId"] != "user-id" {
		t.Fatalf("scope.userRegisteredId = %v, want user-id", scope["userRegisteredId"])
	}
}

func TestEventSpecificIDMatchesEventID(t *testing.T) {
	changedAt := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	passwordChanged := NewPasswordChangedEvent("password-changed-id", changedAt, "user-id", "hash", nil)
	if passwordChanged.EventID != passwordChanged.Data["passwordChangedId"] {
		t.Fatalf("password changed event id = %q, passwordChangedId = %v", passwordChanged.EventID, passwordChanged.Data["passwordChangedId"])
	}

	subjectKey, err := protectedpii.GenerateSubjectDataKey()
	if err != nil {
		t.Fatal(err)
	}
	userNameChanged := NewUserNameChangedEvent("name-changed-id", "Ada Lovelace", changedAt, "user-id", subjectKey, nil)
	if userNameChanged.EventID != userNameChanged.Data["userNameChangedId"] {
		t.Fatalf("name changed event id = %q, userNameChangedId = %v", userNameChanged.EventID, userNameChanged.Data["userNameChangedId"])
	}
}
