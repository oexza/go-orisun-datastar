package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/email"
)

type EmailSender interface {
	Send(ctx context.Context, message email.Message) error
}
