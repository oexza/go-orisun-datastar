package views

import (
	"fmt"
	"os"
	"time"
)

type User struct {
	ID               string
	UserRegisteredID string
	Name             string
	Username         string
	Email            string
	EmailVerified    bool
	Image            string
	Bio              string
	HeaderImageURL   string
}

type Todo struct {
	TodoID    string
	Title     string
	Completed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Field struct {
	Name  string
	Type  string
	Label string
}

func displayName(user User) string {
	if user.Name != "" {
		return user.Name
	}
	if user.Username != "" {
		return user.Username
	}
	return user.Email
}

func postFormSSE(path string) string {
	return fmt.Sprintf("@post('%s', {contentType: 'form'})", path)
}

func longRunningGetSSE(path string) string {
	return fmt.Sprintf("@get('%s', {requestCancellation: 'disabled', retryMaxCount: 1000, retryInterval: 1000, retryMaxWaitMs: 5000})", path)
}

func hotReloadSSE() string {
	return "@get('/reload', {retryMaxCount: 1000, retryInterval: 20, retryMaxWaitMs: 200})"
}

func isDevelopment() bool {
	return os.Getenv("NODE_ENV") != "production"
}

func sendValidationOTPSSE(userID string) string {
	return postFormSSE("/register/" + userID + "/send-email-validation-otp")
}

func todoToggleSSE(todo Todo) string {
	action := "complete"
	if todo.Completed {
		action = "reopen"
	}
	return postFormSSE("/todos/" + todo.TodoID + "/" + action)
}

func todoRenameSSE(todo Todo) string {
	return postFormSSE("/todos/" + todo.TodoID + "/rename")
}

func todoDeleteSSE(todo Todo) string {
	return postFormSSE("/todos/" + todo.TodoID + "/delete")
}
