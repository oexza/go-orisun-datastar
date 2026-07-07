package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/commandlimits"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

type ValidateEmailVerificationOTPCommand struct {
	User     views.User
	Code     string
	Metadata CommandMetadata
}

func ValidateEmailVerificationOTPCommandHandler(ctx context.Context, command ValidateEmailVerificationOTPCommand, saver eventstore.Saver, retriever eventstore.Retriever) error {
	if err := commandlimits.Assert(command); err != nil {
		return err
	}
	model, err := loadValidateEmailVerificationOTPContext(ctx, command, retriever)
	if err != nil {
		return err
	}
	if !model.userExists {
		return errors.New("registered user event not found")
	}
	if model.otpID == "" {
		return errors.New("no verification code found")
	}
	if model.code != strings.TrimSpace(command.Code) || !model.expiresAt.After(time.Now()) {
		return errors.New("invalid or expired verification code")
	}
	if model.alreadyValidated {
		return errors.New("verification code already validated")
	}

	validationID := uuidv7.NewString()
	event := NewEmailVerificationOTPValidatedEvent(validationID, time.Now(), model.otpID, command.User.UserRegisteredID, nil)
	_, err = eventstore.SaveCommandEvents(ctx, saver, command.Metadata, []eventstore.DomainEvent{event}, model.position, model.events, model.query)
	return err
}

type validateEmailVerificationOTPContext struct {
	userExists       bool
	otpID            string
	code             string
	expiresAt        time.Time
	alreadyValidated bool
	position         eventstore.Position
	events           []eventstore.ResolvedEvent
	query            eventstore.Query
}

type latestOTP struct {
	id        string
	code      string
	expiresAt time.Time
	position  eventstore.Position
}

func latestEmailVerificationOTP(events []eventstore.ResolvedEvent) latestOTP {
	otp := latestOTP{position: eventstore.NoEventPosition}
	for _, resolved := range events {
		if !resolved.Position.After(otp.position) {
			continue
		}
		expiresAt, _ := resolved.Event.Data[EmailVerificationOTPExpiresAtField].(string)
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			continue
		}
		otp.id, _ = resolved.Event.Data[EmailVerificationOTPGeneratedIDField].(string)
		otp.code, _ = resolved.Event.Data[EmailVerificationOTPCodeField].(string)
		otp.expiresAt = parsed
		otp.position = resolved.Position
	}
	return otp
}

func loadValidateEmailVerificationOTPContext(ctx context.Context, command ValidateEmailVerificationOTPCommand, retriever eventstore.Retriever) (*validateEmailVerificationOTPContext, error) {
	generatedQuery := emailVerificationOTPGeneratedByUserQuery(command.User.UserRegisteredID)
	userQuery := userRegisteredQuery(command.User.UserRegisteredID)

	var (
		events []eventstore.ResolvedEvent
		latest eventstore.LatestByCriteriaResult
		otp    latestOTP
		query  eventstore.Query
		otpID  string
		stable bool
	)
	for range 5 {
		validationQuery := emailVerificationOTPValidatedQuery(otpID)
		query = combineQueries(generatedQuery, validationQuery, userQuery)
		var err error
		latest, err = retriever.GetLatestByCriteria(ctx, query.Criteria)
		if err != nil {
			return nil, err
		}
		events = eventstore.EventsFromLatest(latest.Results)
		otp = latestEmailVerificationOTP(events)
		if otp.id == otpID || otp.id == "" {
			stable = true
			break
		}
		otpID = otp.id
	}
	if !stable {
		return nil, eventstore.ErrConflict
	}

	model := &validateEmailVerificationOTPContext{
		otpID:     otp.id,
		code:      otp.code,
		expiresAt: otp.expiresAt,
		position:  latest.ContextPosition,
		events:    events,
		query:     query,
	}
	for _, event := range events {
		model.handle(event)
	}
	return model, nil
}

func (m *validateEmailVerificationOTPContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case UserRegistered:
		m.userExists = true
	case EmailVerificationOTPValidated:
		m.alreadyValidated = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}
