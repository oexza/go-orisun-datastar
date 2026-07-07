package email

import (
	"context"
	"log/slog"
)

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type Message struct {
	To    string
	Title string
	Body  string
}

type LogSender struct {
	Logger *slog.Logger
}

func (s LogSender) Send(ctx context.Context, message Message) error {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("email message",
		"to", message.To,
		"title", message.Title,
		"body", message.Body,
	)
	return nil
}
