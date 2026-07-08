package views

import (
	"fmt"
	"time"
)

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
