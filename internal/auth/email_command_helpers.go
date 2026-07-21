package auth

import "github.com/OrisunLabs/go-orisun-datastar/internal/eventstore"

func userRegisteredQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: UserRegistered},
		{Key: UserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func userRegisteredByUsernameOrEmailQuery(usernameHash, emailHash string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{
			{Key: "eventType", Value: UserRegistered},
			{Key: UserRegisteredUsernameHashField, Value: usernameHash},
		}},
		{Tags: []eventstore.Tag{
			{Key: "eventType", Value: UserRegistered},
			{Key: UserRegisteredEmailHashField, Value: emailHash},
		}},
	}}
}

func accountDeletionRequestedByUserQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: AccountDeletionRequested},
		{Key: ScopeUserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func accountDeletedByRequestQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: AccountDeleted},
		{Key: "scope.accountDeletionRequestedId", Value: requestID},
	}}}}
}

func accountDeletedByUserQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: AccountDeleted},
		{Key: ScopeUserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func emailVerificationOTPGeneratedQuery(otpID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: EmailVerificationOTPGeneratedIDField, Value: otpID},
	}}}}
}

func emailVerificationOTPGeneratedByUserQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: ScopeUserRegisteredIDField, Value: userRegisteredID},
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

func passwordResetCompletedQuery(requestID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordResetCompleted},
		{Key: ScopePasswordResetRequestedIDField, Value: requestID},
	}}}}
}

func passwordChangedByUserQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: PasswordChanged},
		{Key: ScopeUserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func userNameChangedByUserQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: UserNameChanged},
		{Key: ScopeUserRegisteredIDField, Value: userRegisteredID},
	}}}}
}

func emailVerificationOTPStateQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPGenerated}, {Key: ScopeUserRegisteredIDField, Value: userRegisteredID}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}, {Key: ScopeUserRegisteredIDField, Value: userRegisteredID}}},
	}}
}

func combineQueries(queries ...eventstore.Query) eventstore.Query {
	combined := eventstore.Query{}
	for _, query := range queries {
		combined.Criteria = append(combined.Criteria, query.Criteria...)
	}
	return combined
}
