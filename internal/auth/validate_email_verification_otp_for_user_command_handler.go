package auth

import (
	"context"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type AuthUserByIDReader interface {
	UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error)
}

func ValidateEmailVerificationOTPForUserCommandHandler(ctx context.Context, userID, code string, metadata CommandMetadata, users AuthUserByIDReader, saver eventstore.Saver, retriever eventstore.Retriever) error {
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
