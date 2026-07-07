package views

import (
	"fmt"
	"os"
	"strings"
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

type TodoStats struct {
	Total      int
	Active     int
	Completed  int
	Completion int
}

type Field struct {
	Name  string
	Type  string
	Label string
}

type OpsDashboardSnapshot struct {
	Handlers []OpsEventHandlerMetric `json:"handlers"`
}

type OpsEventHandlerMetric struct {
	Name                 string `json:"name"`
	Running              bool   `json:"running"`
	LastEventID          string `json:"lastEventId"`
	LastEventType        string `json:"lastEventType"`
	LastCommitPosition   int64  `json:"lastCommitPosition"`
	LastCheckpointCommit int64  `json:"lastCheckpointCommit"`
	ProcessedCount       int64  `json:"processedCount"`
	FailureCount         int64  `json:"failureCount"`
	LastError            string `json:"lastError"`
	UpdatedAt            string `json:"updatedAt"`
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

func userInitials(user User) string {
	name := strings.TrimSpace(displayName(user))
	if name == "" {
		return "?"
	}

	parts := strings.Fields(name)
	if len(parts) == 1 {
		return strings.ToUpper(string([]rune(parts[0])[0]))
	}

	first := []rune(parts[0])
	last := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(first[0]) + string(last[0]))
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

func todoCompleteActiveSSE() string {
	return postFormSSE("/todos/bulk/complete-active")
}

func todoReopenCompletedSSE() string {
	return postFormSSE("/todos/bulk/reopen-completed")
}

func todoClearCompletedSSE() string {
	return postFormSSE("/todos/bulk/clear-completed")
}

func todoStats(todos []Todo) TodoStats {
	stats := TodoStats{Total: len(todos)}
	for _, todo := range todos {
		if todo.Completed {
			stats.Completed++
			continue
		}
		stats.Active++
	}
	if stats.Total > 0 {
		stats.Completion = stats.Completed * 100 / stats.Total
	}
	return stats
}

func todoProgressStyle(stats TodoStats) string {
	return fmt.Sprintf("width: %d%%", stats.Completion)
}

func todoProgressCustomProperty(stats TodoStats) string {
	return fmt.Sprintf("--progress: %d%%", stats.Completion)
}

func todoCountLabel(count int) string {
	return fmt.Sprint(count)
}

func int64Label(count int64) string {
	return fmt.Sprint(count)
}

func todoCompletionLabel(stats TodoStats) string {
	return fmt.Sprintf("%d%%", stats.Completion)
}

func todoTimeLabel(todo Todo) string {
	timestamp := todo.CreatedAt
	label := "Created"
	if todo.UpdatedAt.After(todo.CreatedAt.Add(time.Second)) {
		timestamp = todo.UpdatedAt
		label = "Updated"
	}
	if timestamp.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s %s", label, timestamp.Format("Jan 2, 3:04 PM"))
}
