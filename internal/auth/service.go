package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/views"
)

const (
	UserRegistered                = "UserRegistered"
	UserNameChanged               = "UserNameChanged"
	EmailVerificationOTPGenerated = "EmailVerificationOTPGenerated"
	EmailVerificationOTPValidated = "EmailVerificationOTPValidated"
	EmailVerificationOTPSent      = "EmailVerificationOTPSent"
	PasswordResetRequested        = "PasswordResetRequested"
	PasswordResetEmailSent        = "PasswordResetEmailSent"
	PasswordResetCompleted        = "PasswordResetCompleted"
	PasswordChanged               = "PasswordChanged"
)

type Service struct {
	db            *pgxpool.Pool
	store         eventstore.Saver
	retriever     eventstore.Retriever
	secureCookie  bool
	sessionCookie string
}

func NewService(db *pgxpool.Pool, saver eventstore.Saver, retriever eventstore.Retriever, secureCookie bool) *Service {
	return &Service{
		db:            db,
		store:         saver,
		retriever:     retriever,
		secureCookie:  secureCookie,
		sessionCookie: "go-event-starter-session",
	}
}

type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	FirstName   string
	LastName    string
	YearOfBirth int
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (views.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Username) < 4 || input.Email == "" || len(input.Password) < 6 {
		return views.User{}, errors.New("invalid registration input")
	}

	query := eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}, {Key: "username", Value: input.Username}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}, {Key: "email", Value: input.Email}}},
	}}
	existing, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, query)
	if err != nil {
		return views.User{}, err
	}
	if len(existing) > 0 {
		return views.User{}, errors.New("user already exists")
	}

	userRegisteredID := uuid.NewString()
	userID := uuid.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return views.User{}, err
	}

	event := eventstore.DomainEvent{
		EventID:   userRegisteredID,
		EventType: UserRegistered,
		Data: map[string]any{
			"userRegisteredId": userRegisteredID,
			"username":         input.Username,
			"email":            input.Email,
			"firstName":        strings.TrimSpace(input.FirstName),
			"lastName":         strings.TrimSpace(input.LastName),
			"yearOfBirth":      input.YearOfBirth,
			"scope":            map[string]any{},
		},
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, query); err != nil {
		return views.User{}, err
	}

	name := strings.TrimSpace(input.FirstName + " " + input.LastName)
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_user (id, name, email, email_verified, username, display_username, user_registered_id)
		VALUES ($1, $2, $3, false, $4, $4, $5)
	`, userID, name, input.Email, input.Username, userRegisteredID)
	if err != nil {
		return views.User{}, err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_account (id, account_id, provider_id, user_id, password)
		VALUES ($1, $2, 'credential', $3, $4)
	`, uuid.NewString(), input.Email, userID, string(hash))
	if err != nil {
		return views.User{}, err
	}

	user := views.User{ID: userID, UserRegisteredID: userRegisteredID, Name: name, Username: input.Username, Email: input.Email}
	return user, nil
}

func (s *Service) Login(ctx context.Context, emailAddress, password string) (views.User, string, error) {
	user, hash, err := s.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
	if err != nil {
		return views.User{}, "", errors.New("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return views.User{}, "", errors.New("invalid email or password")
	}
	token, err := randomToken(32)
	if err != nil {
		return views.User{}, "", err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_session (id, token, user_id, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), token, user.ID, time.Now().Add(90*24*time.Hour))
	return user, token, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM auth_session WHERE token = $1`, token)
	return err
}

func (s *Service) CurrentUser(ctx context.Context, r *http.Request) (views.User, bool, error) {
	cookie, err := r.Cookie(s.sessionCookie)
	if err != nil || cookie.Value == "" {
		return views.User{}, false, nil
	}
	user, err := s.UserBySessionToken(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return views.User{}, false, nil
		}
		return views.User{}, false, err
	}
	return user, true, nil
}

func (s *Service) UserBySessionToken(ctx context.Context, token string) (views.User, error) {
	return scanUser(s.db.QueryRow(ctx, `
		SELECT u.id, u.user_registered_id, u.name, coalesce(u.username, ''), u.email, u.email_verified, coalesce(u.image, ''), coalesce(p.bio, ''), coalesce(p.header_image_url, '')
		FROM auth_session s
		JOIN auth_user u ON u.id = s.user_id
		LEFT JOIN profile_stats p ON p.user_id = u.user_registered_id
		WHERE s.token = $1 AND s.expires_at > now()
	`, token))
}

func (s *Service) UserByRegisteredID(ctx context.Context, userRegisteredID string) (views.User, error) {
	return scanUser(s.db.QueryRow(ctx, `
		SELECT u.id, u.user_registered_id, u.name, coalesce(u.username, ''), u.email, u.email_verified, coalesce(u.image, ''), coalesce(p.bio, ''), coalesce(p.header_image_url, '')
		FROM auth_user u
		LEFT JOIN profile_stats p ON p.user_id = u.user_registered_id
		WHERE u.user_registered_id = $1
	`, userRegisteredID))
}

func (s *Service) UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error) {
	return scanUser(s.db.QueryRow(ctx, `
		SELECT u.id, u.user_registered_id, u.name, coalesce(u.username, ''), u.email, u.email_verified, coalesce(u.image, ''), coalesce(p.bio, ''), coalesce(p.header_image_url, '')
		FROM auth_user u
		LEFT JOIN profile_stats p ON p.user_id = u.user_registered_id
		WHERE u.id = $1 OR u.user_registered_id = $1
	`, id))
}

func (s *Service) GenerateEmailVerificationOTP(ctx context.Context, user views.User) error {
	return s.generateEmailVerificationOTP(ctx, user, nil)
}

func (s *Service) GenerateEmailVerificationOTPWithMetadata(ctx context.Context, user views.User, metadata CommandMetadata) error {
	return s.generateEmailVerificationOTP(ctx, user, metadata)
}

func (s *Service) generateEmailVerificationOTP(ctx context.Context, user views.User, metadata CommandMetadata) error {
	model, err := s.emailVerificationOTPContext(ctx, user.UserRegisteredID)
	if err != nil {
		return err
	}
	if model.emailValidated {
		return nil
	}
	if model.latestOTPExpiresAt.After(time.Now()) {
		return nil
	}

	code, err := numericCode(6)
	if err != nil {
		return err
	}
	otpID := uuid.NewString()
	expiresAt := time.Now().Add(15 * time.Minute)
	query := emailVerificationOTPGeneratedQuery(otpID)
	event := eventstore.DomainEvent{
		EventID:   otpID,
		EventType: EmailVerificationOTPGenerated,
		Data: map[string]any{
			"emailVerificationOTPGeneratedId": otpID,
			"otpCode":                         code,
			"expiresAt":                       expiresAt.Format(time.RFC3339),
			"scope":                           map[string]any{"userRegisteredId": user.UserRegisteredID},
		},
		Metadata: metadataWithQuery(metadata, query),
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_verification (id, identifier, value, expires_at)
		VALUES ($1, $2, $3, $4)
	`, otpID, "email:"+user.UserRegisteredID, code, expiresAt)
	if err != nil {
		return err
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, query); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error {
	_, err := s.db.Exec(ctx, `UPDATE auth_user SET image = $1, updated_at = now() WHERE user_registered_id = $2`, imageURL, userRegisteredID)
	return err
}

func (s *Service) MarkEmailVerified(ctx context.Context, userRegisteredID string) error {
	_, err := s.db.Exec(ctx, `UPDATE auth_user SET email_verified = true, updated_at = now() WHERE user_registered_id = $1`, userRegisteredID)
	return err
}

func (s *Service) ValidateOTP(ctx context.Context, userID, code string) error {
	user, err := s.UserByIDOrRegisteredID(ctx, userID)
	if err != nil {
		return err
	}

	generatedQuery := eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: "scope.userRegisteredId", Value: user.UserRegisteredID},
	}}}}
	generatedEvents, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 100, eventstore.Forward, generatedQuery)
	if err != nil {
		return err
	}
	otp := latestEmailVerificationOTP(generatedEvents)
	if otp.id == "" {
		return errors.New("no verification code found")
	}
	if otp.code != strings.TrimSpace(code) || !otp.expiresAt.After(time.Now()) {
		return errors.New("invalid or expired verification code")
	}

	validationQuery := emailVerificationOTPValidatedQuery(otp.id)
	validationEvents, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, validationQuery)
	if err != nil {
		return err
	}
	if len(validationEvents) > 0 {
		return errors.New("verification code already validated")
	}

	validationID := uuid.NewString()
	event := eventstore.DomainEvent{
		EventID:   validationID,
		EventType: EmailVerificationOTPValidated,
		Data: map[string]any{
			"emailVerificationOTPValidatedId": validationID,
			"validatedAt":                     time.Now().Format(time.RFC3339),
			"scope":                           map[string]any{"emailVerificationOTPGeneratedId": otp.id, "userRegisteredId": user.UserRegisteredID},
		},
		Metadata: metadataWithQuery(nil, combineQueries(generatedQuery, validationQuery)),
	}
	modelPosition := eventstore.NoEventPosition
	handledEvents := append(generatedEvents, validationEvents...)
	for _, resolved := range handledEvents {
		if resolved.Position.After(modelPosition) {
			modelPosition = resolved.Position
		}
	}
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, modelPosition, handledEvents, combineQueries(generatedQuery, validationQuery))
	return err
}

func (s *Service) RequestPasswordReset(ctx context.Context, emailAddress string) error {
	user, _, err := s.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
	if err != nil {
		return nil
	}
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	requestID := uuid.NewString()
	expiresAt := time.Now().Add(30 * time.Minute)
	event := eventstore.DomainEvent{
		EventID:   requestID,
		EventType: PasswordResetRequested,
		Data: map[string]any{
			"passwordResetRequestedId": requestID,
			"email":                    user.Email,
			"resetToken":               token,
			"expiresAt":                expiresAt.Format(time.RFC3339),
			"scope":                    map[string]any{"userRegisteredId": user.UserRegisteredID},
		},
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO auth_verification (id, identifier, value, expires_at)
		VALUES ($1, $2, $3, $4)
	`, requestID, "password-reset:"+user.ID, token, expiresAt)
	if err != nil {
		return err
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, passwordResetRequestedQuery(requestID)); err != nil {
		return err
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	var requestID, identifier string
	err := s.db.QueryRow(ctx, `
		SELECT id, identifier FROM auth_verification
		WHERE identifier LIKE 'password-reset:%' AND value = $1 AND expires_at > now()
		ORDER BY created_at DESC LIMIT 1
	`, token).Scan(&requestID, &identifier)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}
	userID := strings.TrimPrefix(identifier, "password-reset:")
	if err := s.setPassword(ctx, userID, password); err != nil {
		return err
	}
	event := eventstore.DomainEvent{EventID: uuid.NewString(), EventType: PasswordResetCompleted, Data: map[string]any{"passwordResetCompletedId": uuid.NewString(), "completedAt": time.Now().Format(time.RFC3339), "scope": map[string]any{"passwordResetRequestedId": requestID}}}
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{})
	return err
}

type emailVerificationOTPModel struct {
	emailValidated     bool
	latestOTPExpiresAt time.Time
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
		expiresAt, _ := resolved.Event.Data["expiresAt"].(string)
		parsed, err := time.Parse(time.RFC3339, expiresAt)
		if err != nil {
			continue
		}
		otp.id, _ = resolved.Event.Data["emailVerificationOTPGeneratedId"].(string)
		otp.code, _ = resolved.Event.Data["otpCode"].(string)
		otp.expiresAt = parsed
		otp.position = resolved.Position
	}
	return otp
}

func (s *Service) emailVerificationOTPContext(ctx context.Context, userRegisteredID string) (emailVerificationOTPModel, error) {
	query := eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPGenerated}, {Key: "scope.userRegisteredId", Value: userRegisteredID}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}, {Key: "scope.userRegisteredId", Value: userRegisteredID}}},
	}}
	events, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 20, eventstore.Forward, query)
	if err != nil {
		return emailVerificationOTPModel{}, err
	}
	model := emailVerificationOTPModel{}
	for _, resolved := range events {
		switch resolved.Event.EventType {
		case EmailVerificationOTPGenerated:
			expiresAt, _ := resolved.Event.Data["expiresAt"].(string)
			parsed, err := time.Parse(time.RFC3339, expiresAt)
			if err == nil && parsed.After(model.latestOTPExpiresAt) {
				model.latestOTPExpiresAt = parsed
			}
		case EmailVerificationOTPValidated:
			model.emailValidated = true
		}
	}
	return model, nil
}

func (s *Service) ChangePassword(ctx context.Context, user views.User, currentPassword, newPassword string) error {
	_, hash, err := s.userByEmailWithPassword(ctx, user.Email)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	if err := s.setPassword(ctx, user.ID, newPassword); err != nil {
		return err
	}
	event := eventstore.DomainEvent{EventID: uuid.NewString(), EventType: PasswordChanged, Data: map[string]any{"passwordChangedId": uuid.NewString(), "changedAt": time.Now().Format(time.RFC3339), "scope": map[string]any{"userRegisteredId": user.UserRegisteredID}}}
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{})
	return err
}

func (s *Service) UpdateName(ctx context.Context, user views.User, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	event := eventstore.DomainEvent{EventID: uuid.NewString(), EventType: UserNameChanged, Data: map[string]any{"userNameChangedId": uuid.NewString(), "name": name, "changedAt": time.Now().Format(time.RFC3339), "scope": map[string]any{"userRegisteredId": user.UserRegisteredID}}}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{}); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `UPDATE auth_user SET name = $1, updated_at = now() WHERE id = $2`, name, user.ID)
	return err
}

func (s *Service) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie, Expires: time.Now().Add(90 * 24 * time.Hour)})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookie, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie, MaxAge: -1})
}

func (s *Service) SessionCookieName() string {
	return s.sessionCookie
}

func (s *Service) setPassword(ctx context.Context, userID, password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `UPDATE auth_account SET password = $1, updated_at = now() WHERE user_id = $2 AND provider_id = 'credential'`, string(hash), userID)
	return err
}

func (s *Service) userByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
	var user views.User
	var hash string
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.user_registered_id, u.name, coalesce(u.username, ''), u.email, u.email_verified, coalesce(u.image, ''), coalesce(p.bio, ''), coalesce(p.header_image_url, ''), a.password
		FROM auth_user u
		JOIN auth_account a ON a.user_id = u.id AND a.provider_id = 'credential'
		LEFT JOIN profile_stats p ON p.user_id = u.user_registered_id
		WHERE u.email = $1
	`, emailAddress).Scan(&user.ID, &user.UserRegisteredID, &user.Name, &user.Username, &user.Email, &user.EmailVerified, &user.Image, &user.Bio, &user.HeaderImageURL, &hash)
	return user, hash, err
}

func scanUser(row pgx.Row) (views.User, error) {
	var user views.User
	err := row.Scan(&user.ID, &user.UserRegisteredID, &user.Name, &user.Username, &user.Email, &user.EmailVerified, &user.Image, &user.Bio, &user.HeaderImageURL)
	return user, err
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func numericCode(size int) (string, error) {
	var b strings.Builder
	for i := 0; i < size; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String(), nil
}
