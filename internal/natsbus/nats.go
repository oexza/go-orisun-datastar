package natsbus

import (
	"context"
	"encoding/json"

	"github.com/example/go-orisun-datastar/internal/eventstore"
	"github.com/nats-io/nats.go"
)

type Bus struct {
	conn *nats.Conn
}

func Connect(url string) (*Bus, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Bus{conn: conn}, nil
}

func (b *Bus) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
}

func (b *Bus) Publish(ctx context.Context, subject string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return b.conn.Publish(subject, payload)
}

func (b *Bus) Subscribe(ctx context.Context, subject string, handle func(context.Context, []byte)) (eventstore.MessageSubscription, error) {
	sub, err := b.conn.Subscribe(subject, func(msg *nats.Msg) {
		handle(ctx, msg.Data)
	})
	if err != nil {
		return nil, err
	}
	return &Subscription{sub: sub}, nil
}

type Subscription struct {
	sub *nats.Subscription
}

func (s *Subscription) Close() error {
	return s.sub.Unsubscribe()
}
