package auth

import (
	"context"
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/uuidv7"
)

type RecordLoginAttemptCommand struct {
	AttemptedIdentifier string
	IPAddress           string
	UserRegisteredID    string
	Succeeded           bool
	Metadata            CommandMetadata
}

func RecordLoginAttemptCommandHandler(ctx context.Context, command RecordLoginAttemptCommand, saver eventstore.Saver) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	eventID := uuidv7.NewString()
	event := NewLoginAttemptRecordedEvent(eventID, time.Now(), command.AttemptedIdentifier, command.IPAddress, command.UserRegisteredID, command.Succeeded, nil)
	_, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, eventstore.LastEventPosition, nil, loginAttemptRecordedQuery(eventID))
	return err
}

func loginAttemptRecordedQuery(eventID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: LoginAttemptRecorded},
		{Key: LoginAttemptRecordedIDField, Value: eventID},
	}}}}
}
