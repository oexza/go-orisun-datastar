package eventstore

import (
	"context"
	"log/slog"
)

type Projector struct {
	Name         string
	Store        Subscriber
	Checkpointer Checkpointer
	Query        Query
	Logger       *slog.Logger
	Handle       func(context.Context, ResolvedEvent) error
}

func (p Projector) Start(ctx context.Context) error {
	position, ok, err := p.Checkpointer.GetCheckpoint(ctx, p.Name)
	if err != nil {
		return err
	}
	if !ok {
		position = NoEventPosition
	}
	return p.Store.SubscribeToEvents(ctx, p.Name, position, p.Query, func(ctx context.Context, event ResolvedEvent) error {
		if err := p.Handle(ctx, event); err != nil {
			return err
		}
		return p.Checkpointer.UpdateCheckpoint(ctx, p.Name, event.Position)
	})
}
