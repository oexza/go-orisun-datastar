package profile

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type UpdateProfileBioCommand struct {
	User     views.User
	Bio      string
	Metadata map[string]any
}

func UpdateProfileBioCommandHandler(ctx context.Context, command UpdateProfileBioCommand, saver eventstore.Saver) error {
	bio := strings.TrimSpace(command.Bio)
	if len(bio) > 280 {
		return errors.New("bio must be 280 characters or fewer")
	}
	eventID := uuidv7.NewString()
	event := NewProfileBioUpdatedEvent(eventID, bio, time.Now(), command.User.UserRegisteredID, command.Metadata)
	_, err := saver.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, profileEventQuery(ProfileBioUpdated, ProfileBioUpdatedIDField, eventID))
	return err
}
