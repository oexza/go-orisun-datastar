package profile

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type EventSaver interface {
	SaveEvents(ctx context.Context, events []eventstore.DomainEvent, expected eventstore.Position, scopeEvents []eventstore.ResolvedEvent, subset eventstore.Query) (eventstore.WriteResult, error)
}

type EventSubscriber interface {
	SubscribeToEvents(ctx context.Context, subscriberName string, after eventstore.Position, query eventstore.Query, handle func(context.Context, eventstore.ResolvedEvent) error) error
}

type EventCheckpointer interface {
	GetCheckpoint(ctx context.Context, name string) (eventstore.Position, bool, error)
	UpdateCheckpoint(ctx context.Context, name string, position eventstore.Position) error
}

type ObjectStore interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	PublicURL(key string) string
}
