package todo

import (
	"context"
	"time"

	"github.com/example/hono-event-starter-go/internal/appdb"
	"github.com/example/hono-event-starter-go/internal/eventstore"
)

type Projector struct {
	db        *appdb.DB
	publisher eventstore.Publisher
}

func NewProjector(db *appdb.DB, publisher eventstore.Publisher) *Projector {
	return &Projector{db: db, publisher: publisher}
}

func Query() eventstore.Query {
	criteria := make([]eventstore.Criterion, 0, 5)
	for _, eventType := range []string{TodoCreated, TodoRenamed, TodoCompleted, TodoReopened, TodoDeleted} {
		criteria = append(criteria, eventstore.Criterion{Tags: []eventstore.Tag{{Key: "eventType", Value: eventType}}})
	}
	return eventstore.Query{Criteria: criteria}
}

func (p *Projector) Handle(ctx context.Context, resolved eventstore.ResolvedEvent) error {
	data := resolved.Event.Data
	scope := eventstore.Scope(data)
	todoID, _ := scope["todoId"].(string)
	userRegisteredID, _ := scope["userRegisteredId"].(string)
	var err error
	switch resolved.Event.EventType {
	case TodoCreated:
		createdAt := parseTime(data["createdAt"])
		title, _ := data["title"].(string)
		_, err = p.db.Exec(ctx, `
			INSERT INTO todo_items (todo_id, user_registered_id, title, completed, completed_at, deleted_at, last_event_commit_position, last_event_prepare_position, created_at, updated_at)
			VALUES ($1, $2, $3, false, null, null, $4, $5, $6, $6)
			ON CONFLICT (todo_id) DO NOTHING
		`, todoID, userRegisteredID, title, resolved.Position.Commit, resolved.Position.Prepare, createdAt)
	case TodoRenamed:
		renamedAt := parseTime(data["renamedAt"])
		title, _ := data["title"].(string)
		_, err = p.db.Exec(ctx, `UPDATE todo_items SET title = $1, last_event_commit_position = $2, last_event_prepare_position = $3, updated_at = $4 WHERE todo_id = $5`, title, resolved.Position.Commit, resolved.Position.Prepare, renamedAt, todoID)
	case TodoCompleted:
		completedAt := parseTime(data["completedAt"])
		_, err = p.db.Exec(ctx, `UPDATE todo_items SET completed = true, completed_at = $1, last_event_commit_position = $2, last_event_prepare_position = $3, updated_at = $1 WHERE todo_id = $4`, completedAt, resolved.Position.Commit, resolved.Position.Prepare, todoID)
	case TodoReopened:
		reopenedAt := parseTime(data["reopenedAt"])
		_, err = p.db.Exec(ctx, `UPDATE todo_items SET completed = false, completed_at = null, last_event_commit_position = $1, last_event_prepare_position = $2, updated_at = $3 WHERE todo_id = $4`, resolved.Position.Commit, resolved.Position.Prepare, reopenedAt, todoID)
	case TodoDeleted:
		deletedAt := parseTime(data["deletedAt"])
		_, err = p.db.Exec(ctx, `UPDATE todo_items SET deleted_at = $1, last_event_commit_position = $2, last_event_prepare_position = $3, updated_at = $1 WHERE todo_id = $4`, deletedAt, resolved.Position.Commit, resolved.Position.Prepare, todoID)
	}
	if err != nil {
		return err
	}
	if userRegisteredID != "" {
		return p.publisher.Publish(ctx, Channel(userRegisteredID), map[string]string{"userRegisteredId": userRegisteredID})
	}
	return nil
}

func parseTime(value any) time.Time {
	text, _ := value.(string)
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Now()
	}
	return parsed
}
