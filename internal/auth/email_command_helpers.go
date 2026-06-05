package auth

import "github.com/oexza/go-orisun-datastar/internal/eventstore"

type emailValidationOTPContext struct {
	otpID       string
	code        string
	expiresAt   string
	email       string
	alreadySent bool
	position    eventstore.Position
}

func (m *emailValidationOTPContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case EmailVerificationOTPGenerated:
		m.otpID, _ = resolved.Event.Data["emailVerificationOTPGeneratedId"].(string)
		m.code, _ = resolved.Event.Data["otpCode"].(string)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case UserRegistered:
		m.email, _ = resolved.Event.Data["email"].(string)
	case EmailVerificationOTPSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}

type passwordResetEmailContext struct {
	requestID   string
	email       string
	token       string
	expiresAt   string
	alreadySent bool
	position    eventstore.Position
}

func (m *passwordResetEmailContext) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case PasswordResetRequested:
		m.requestID, _ = resolved.Event.Data["passwordResetRequestedId"].(string)
		m.email, _ = resolved.Event.Data["email"].(string)
		m.token, _ = resolved.Event.Data["resetToken"].(string)
		m.expiresAt, _ = resolved.Event.Data["expiresAt"].(string)
	case PasswordResetEmailSent:
		m.alreadySent = true
	}
	if resolved.Position.After(m.position) {
		m.position = resolved.Position
	}
}

func userRegisteredQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: UserRegistered},
		{Key: UserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func emailVerificationOTPGeneratedQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: EmailVerificationOTPGeneratedIDField, Value: otpID},
	}}}}
}

func emailVerificationOTPSentQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPSent},
		{Key: ScopeEmailVerificationOTPGeneratedIDField, Value: otpID},
	}}}}
}

func emailVerificationOTPValidatedQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPValidated},
		{Key: ScopeEmailVerificationOTPGeneratedIDField, Value: otpID},
	}}}}
}

func passwordResetRequestedQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordResetRequested},
		{Key: PasswordResetRequestedIDField, Value: requestID},
	}}}}
}

func passwordResetEmailSentQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordResetEmailSent},
		{Key: ScopePasswordResetRequestedIDField, Value: requestID},
	}}}}
}

func combineQueries(queries ...eventstore.Query) eventstore.Query {
	combined := eventstore.Query{}
	for _, query := range queries {
		combined.Criteria = append(combined.Criteria, query.Criteria...)
	}
	return combined
}

func metadataWithQuery(metadata CommandMetadata, query eventstore.Query) map[string]any {
	return eventstore.MergeMetadata(map[string]any{"query": eventstore.MustJSON(query)}, metadata)
}
