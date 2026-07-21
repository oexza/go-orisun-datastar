package auth

import (
	"context"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
)

type AuthUserByIDReader interface {
	UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error)
}

func ValidateEmailVerificationOTPForUserCommandHandler(ctx context.Context, userID, code string, metadata CommandMetadata, users AuthUserByIDReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(struct {
		UserID string
		Code   string
	}{UserID: userID, Code: code}); err != nil {
		return err
	}
	user, err := users.UserByIDOrRegisteredID(ctx, userID)
	if err != nil {
		return err
	}
	return ValidateEmailVerificationOTPCommandHandler(ctx, ValidateEmailVerificationOTPCommand{
		User:     user,
		Code:     code,
		Metadata: metadata,
	}, saver, retriever)
}
