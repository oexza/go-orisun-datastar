package todo

import (
	"errors"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type CommandMetadata = eventstore.CommandMetadata

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 160 {
		return "", errors.New("todo title must be between 1 and 160 characters")
	}
	return title, nil
}

func streamQuery(todoID, userRegisteredID string) eventstore.Query {
	criteria := make([]eventstore.Criterion, 0, 5)
	for _, eventType := range []string{TodoCreated, TodoRenamed, TodoCompleted, TodoReopened, TodoDeleted} {
		criteria = append(criteria, eventstore.Criterion{Tags: []eventstore.Tag{
			{Key: "eventType", Value: eventType},
			{Key: TodoScopeIDField, Value: todoID},
			{Key: TodoScopeUserRegisteredIDField, Value: userRegisteredID},
		}})
	}
	return eventstore.Query{Criteria: criteria}
}
