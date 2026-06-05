package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/email"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type EventSaver interface {
	SaveEvents(ctx context.Context, events []eventstore.DomainEvent, expected eventstore.Position, scopeEvents []eventstore.ResolvedEvent, subset eventstore.Query) (eventstore.WriteResult, error)
}

type EventRetriever interface {
	GetEvents(ctx context.Context, from eventstore.Position, count int, direction eventstore.Direction, query eventstore.Query) ([]eventstore.ResolvedEvent, error)
}

type EventSubscriber interface {
	SubscribeToEvents(ctx context.Context, subscriberName string, after eventstore.Position, query eventstore.Query, handle func(context.Context, eventstore.ResolvedEvent) error) error
}

type EventCheckpointer interface {
	GetCheckpoint(ctx context.Context, name string) (eventstore.Position, bool, error)
	UpdateCheckpoint(ctx context.Context, name string, position eventstore.Position) error
}

type EmailSender interface {
	Send(ctx context.Context, message email.Message) error
}
