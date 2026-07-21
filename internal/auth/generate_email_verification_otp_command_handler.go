package auth

import (
	"context"
	"errors"
	"time"

	"github.com/OrisunLabs/go-orisun-datastar/internal/commandlimits"
	"github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"
	"github.com/OrisunLabs/go-orisun-datastar/internal/uuidv7"
	"github.com/OrisunLabs/go-orisun-datastar/internal/views"
)

type GenerateEmailVerificationOTPCommand struct {
	User     views.User
	Metadata CommandMetadata
}

type GenerateEmailVerificationOTPResult struct {
	EmailVerificationOTPGeneratedID string
	Code                            string
	ExpiresAt                       time.Time
	Skipped                         bool
}

func GenerateEmailVerificationOTPCommandHandler(ctx context.Context, command GenerateEmailVerificationOTPCommand, saver eventstore.Saver, retriever eventstore.Retriever) (GenerateEmailVerificationOTPResult, error) {
	if err := commandlimits.Assert(command); err != nil {
		return GenerateEmailVerificationOTPResult{}, err
	}
	model, err := loadGenerateEmailVerificationOTPContext(ctx, command, retriever)
	if err != nil {
		return GenerateEmailVerificationOTPResult{}, err
	}
	if !model.userExists {
		return GenerateEmailVerificationOTPResult{}, errors.New("registered user event not found")
	}
	if model.emailValidated || model.latestOTPExpiresAt.After(time.Now()) {
		return GenerateEmailVerificationOTPResult{Skipped: true}, nil
	}

	code, err := numericCode(6)
	if err != nil {
		return GenerateEmailVerificationOTPResult{}, err
	}
	otpID := uuidv7.NewString()
	expiresAt := time.Now().Add(15 * time.Minute)
	event := NewEmailVerificationOTPGeneratedEvent(otpID, code, expiresAt, command.User.UserRegisteredID, nil)
	if _, err := eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query); err != nil {
		return GenerateEmailVerificationOTPResult{}, err
	}
	return GenerateEmailVerificationOTPResult{EmailVerificationOTPGeneratedID: otpID, Code: code, ExpiresAt: expiresAt}, nil
}

type generateEmailVerificationOTPContext struct {
	userExists         bool
	emailValidated     bool
	latestOTPExpiresAt time.Time
	position           eventstore.Position
	events             []eventstore.ResolvedEvent
	query              eventstore.Query
}

func loadGenerateEmailVerificationOTPContext(ctx context.Context, command GenerateEmailVerificationOTPCommand, retriever eventstore.Retriever) (*generateEmailVerificationOTPContext, error) {
	userQuery := userRegisteredQuery(command.User.UserRegisteredID)
	stateQuery := emailVerificationOTPStateQuery(command.User.UserRegisteredID)
	query := combineQueries(userQuery, stateQuery)

	latest, err := retriever.GetLatestByCriteria(ctx, query.Criteria)
	if err != nil {
		return nil, err
	}

	events := eventstore.EventsFromLatest(latest.Results)
	model := &generateEmailVerificationOTPContext{position: latest.ContextPosition, events: events, query: query}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *generateEmailVerificationOTPContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case UserRegistered:
		m.userExists = true
	case EmailVerificationOTPGenerated:
		expiresAt, _ := resolved.Event.Data[EmailVerificationOTPExpiresAtField].(string)
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err == nil {
			m.latestOTPExpiresAt = parsed
		}
	case EmailVerificationOTPValidated:
		m.emailValidated = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
