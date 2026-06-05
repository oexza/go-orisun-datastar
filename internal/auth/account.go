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

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"golang.org/x/crypto/bcrypt"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"zombiezen.com/go/sqlite"
)

type Service struct {
	db            *appdb.DB
	store         eventstore.Saver
	retriever     eventstore.Retriever
	secureCookie  bool
	sessionCookie string
}

func NewService(db *appdb.DB, saver eventstore.Saver, retriever eventstore.Retriever, secureCookie bool) *Service {
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
	Metadata    CommandMetadata
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (views.User, error) {
	return s.RegisterWithMetadata(ctx, input, input.Metadata)
}

func (s *Service) RegisterWithMetadata(ctx context.Context, input RegisterInput, metadata CommandMetadata) (views.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if len(input.Username) < 4 || input.Email == "" || len(input.Password) < 6 {
		return views.User{}, errors.New("invalid registration input")
	}

	query := eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}, {Key: UserRegisteredUsernameField, Value: input.Username}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: UserRegistered}, {Key: UserRegisteredEmailField, Value: input.Email}}},
	}}
	existing, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, query)
	if err != nil {
		return views.User{}, err
	}
	if len(existing) > 0 {
		return views.User{}, errors.New("user already exists")
	}

	userRegisteredID := uuidv7.NewString()
	userID := uuidv7.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return views.User{}, err
	}

	event := NewUserRegisteredEvent(userRegisteredID, input.Username, input.Email, strings.TrimSpace(input.FirstName), strings.TrimSpace(input.LastName), input.YearOfBirth, metadata)
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, query); err != nil {
		return views.User{}, err
	}

	name := strings.TrimSpace(input.FirstName + " " + input.LastName)
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		if err := dbsql.OnceCreateAuthUser(conn, dbsql.CreateAuthUserParams{
			Id:               userID,
			Name:             name,
			Email:            input.Email,
			Username:         stringPtr(input.Username),
			UserRegisteredId: userRegisteredID,
		}); err != nil {
			return err
		}
		return dbsql.OnceCreateAuthAccount(conn, dbsql.CreateAuthAccountParams{
			Id:        uuidv7.NewString(),
			AccountId: input.Email,
			UserId:    userID,
			Password:  stringPtr(string(hash)),
		})
	}); err != nil {
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
	err = s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthSession(conn, dbsql.CreateAuthSessionParams{
			Id:        uuidv7.NewString(),
			Token:     token,
			UserId:    user.ID,
			ExpiresAt: appdb.SQLTime(time.Now().Add(90 * 24 * time.Hour)),
		})
	})
	return user, token, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceDeleteAuthSessionByToken(conn, token)
	})
}

func (s *Service) CurrentUser(ctx context.Context, r *http.Request) (views.User, bool, error) {
	cookie, err := r.Cookie(s.sessionCookie)
	if err != nil || cookie.Value == "" {
		return views.User{}, false, nil
	}
	user, err := s.UserBySessionToken(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, appdb.ErrNoRows) {
			return views.User{}, false, nil
		}
		return views.User{}, false, err
	}
	return user, true, nil
}

func (s *Service) UserBySessionToken(ctx context.Context, token string) (views.User, error) {
	var row *dbsql.UserBySessionTokenRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserBySessionToken(conn, token)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromSessionRow(row)
}

func (s *Service) UserByRegisteredID(ctx context.Context, userRegisteredID string) (views.User, error) {
	var row *dbsql.UserByRegisteredIdRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByRegisteredId(conn, userRegisteredID)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromRegisteredRow(row)
}

func (s *Service) UserByIDOrRegisteredID(ctx context.Context, id string) (views.User, error) {
	var row *dbsql.UserByIdorRegisteredIdRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByIdorRegisteredId(conn, id)
		return err
	}); err != nil {
		return views.User{}, err
	}
	return userFromIDOrRegisteredRow(row)
}

func (s *Service) GenerateEmailVerificationOTP(ctx context.Context, user views.User) error {
	return s.generateEmailVerificationOTP(ctx, user, nil)
}

func (s *Service) GenerateEmailVerificationOTPWithMetadata(ctx context.Context, user views.User, metadata CommandMetadata) error {
	return s.generateEmailVerificationOTP(ctx, user, metadata)
}

func (s *Service) generateEmailVerificationOTP(ctx context.Context, user views.User, metadata CommandMetadata) error {
	userQuery := userRegisteredQuery(user.UserRegisteredID)
	userEvents, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, userQuery)
	if err != nil {
		return err
	}
	if len(userEvents) == 0 {
		return errors.New("registered user event not found")
	}

	latestStateQuery := emailVerificationOTPStateQuery(user.UserRegisteredID)
	latestEvents, err := s.retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, latestStateQuery)
	if err != nil {
		return err
	}

	model := emailVerificationOTPModel{position: eventstore.NoEventPosition}
	for _, resolved := range append(userEvents, latestEvents...) {
		model.handle(resolved)
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
	otpID := uuidv7.NewString()
	expiresAt := time.Now().Add(15 * time.Minute)
	query := emailVerificationOTPGeneratedQuery(otpID)
	event := NewEmailVerificationOTPGeneratedEvent(otpID, code, expiresAt, user.UserRegisteredID, metadataWithQuery(metadata, query))
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         otpID,
			Identifier: "email:" + user.UserRegisteredID,
			Value:      code,
			ExpiresAt:  appdb.SQLTime(expiresAt),
		})
	}); err != nil {
		return err
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, model.position, model.eventsHandled, combineQueries(userQuery, latestStateQuery)); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateImage(ctx context.Context, userRegisteredID, imageURL string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserImage(conn, dbsql.UpdateAuthUserImageParams{
			Image:            stringPtr(imageURL),
			UserRegisteredId: userRegisteredID,
		})
	})
}

func (s *Service) MarkEmailVerified(ctx context.Context, userRegisteredID string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceMarkAuthUserEmailVerified(conn, userRegisteredID)
	})
}

func (s *Service) ValidateOTP(ctx context.Context, userID, code string) error {
	return s.ValidateOTPWithMetadata(ctx, userID, code, nil)
}

func (s *Service) ValidateOTPWithMetadata(ctx context.Context, userID, code string, metadata CommandMetadata) error {
	user, err := s.UserByIDOrRegisteredID(ctx, userID)
	if err != nil {
		return err
	}

	generatedQuery := eventstore.Query{Criteria: []eventstore.Criterion{{Tags: []eventstore.Tag{
		{Key: "eventType", Value: EmailVerificationOTPGenerated},
		{Key: ScopeUserRegisteredIDField, Value: user.UserRegisteredID},
	}}}}
	generatedEvents, err := s.retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, generatedQuery)
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

	userQuery := userRegisteredQuery(user.UserRegisteredID)
	userEvents, err := s.retriever.GetEvents(ctx, eventstore.NoEventPosition, 1, eventstore.Forward, userQuery)
	if err != nil {
		return err
	}
	if len(userEvents) == 0 {
		return errors.New("registered user event not found")
	}

	validationID := uuidv7.NewString()
	event := NewEmailVerificationOTPValidatedEvent(validationID, time.Now(), otp.id, user.UserRegisteredID, metadataWithQuery(metadata, combineQueries(generatedQuery, validationQuery)))
	modelPosition := eventstore.NoEventPosition
	handledEvents := append(append(generatedEvents, validationEvents...), userEvents...)
	for _, resolved := range handledEvents {
		if resolved.Position.After(modelPosition) {
			modelPosition = resolved.Position
		}
	}
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, modelPosition, handledEvents, combineQueries(generatedQuery, validationQuery, userQuery))
	return err
}

func (s *Service) RequestPasswordReset(ctx context.Context, emailAddress string) error {
	return s.RequestPasswordResetWithMetadata(ctx, emailAddress, nil)
}

func (s *Service) RequestPasswordResetWithMetadata(ctx context.Context, emailAddress string, metadata CommandMetadata) error {
	user, _, err := s.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
	if err != nil {
		return nil
	}
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	requestID := uuidv7.NewString()
	expiresAt := time.Now().Add(30 * time.Minute)
	event := NewPasswordResetRequestedEvent(requestID, user.Email, token, expiresAt, user.UserRegisteredID, metadata)
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         requestID,
			Identifier: "password-reset:" + user.ID,
			Value:      token,
			ExpiresAt:  appdb.SQLTime(expiresAt),
		})
	}); err != nil {
		return err
	}
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, passwordResetRequestedQuery(requestID)); err != nil {
		return err
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	return s.ResetPasswordWithMetadata(ctx, token, password, nil)
}

func (s *Service) ResetPasswordWithMetadata(ctx context.Context, token, password string, metadata CommandMetadata) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	var verification *dbsql.PasswordResetVerificationByTokenRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		verification, err = dbsql.OncePasswordResetVerificationByToken(conn, token)
		return err
	}); err != nil || verification == nil {
		return errors.New("invalid or expired reset token")
	}
	userID := strings.TrimPrefix(verification.Identifier, "password-reset:")
	user, err := s.UserByIDOrRegisteredID(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.setPassword(ctx, userID, password); err != nil {
		return err
	}
	passwordResetCompletedID := uuidv7.NewString()
	event := NewPasswordResetCompletedEvent(passwordResetCompletedID, time.Now(), verification.Id, user.UserRegisteredID, metadata)
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{})
	return err
}

type emailVerificationOTPModel struct {
	emailValidated     bool
	latestOTPExpiresAt time.Time
	position           eventstore.Position
	eventsHandled      []eventstore.ResolvedEvent
}

func (m *emailVerificationOTPModel) handle(resolved eventstore.ResolvedEvent) {
	switch resolved.Event.EventType {
	case EmailVerificationOTPGenerated:
		expiresAt, _ := resolved.Event.Data["expiresAt"].(string)
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
	m.eventsHandled = append(m.eventsHandled, resolved)
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
	events, err := s.retriever.GetEvents(ctx, eventstore.LastEventPosition, 1, eventstore.Backward, emailVerificationOTPStateQuery(userRegisteredID))
	if err != nil {
		return emailVerificationOTPModel{}, err
	}
	model := emailVerificationOTPModel{position: eventstore.NoEventPosition}
	for _, resolved := range events {
		model.handle(resolved)
	}
	return model, nil
}

func emailVerificationOTPStateQuery(userRegisteredID string) eventstore.Query {
	return eventstore.Query{Criteria: []eventstore.Criterion{
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPGenerated}, {Key: ScopeUserRegisteredIDField, Value: userRegisteredID}}},
		{Tags: []eventstore.Tag{{Key: "eventType", Value: EmailVerificationOTPValidated}, {Key: ScopeUserRegisteredIDField, Value: userRegisteredID}}},
	}}
}

func (s *Service) ChangePassword(ctx context.Context, user views.User, currentPassword, newPassword string) error {
	return s.ChangePasswordWithMetadata(ctx, user, currentPassword, newPassword, nil)
}

func (s *Service) ChangePasswordWithMetadata(ctx context.Context, user views.User, currentPassword, newPassword string, metadata CommandMetadata) error {
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
	passwordChangedID := uuidv7.NewString()
	event := NewPasswordChangedEvent(passwordChangedID, time.Now(), user.UserRegisteredID, metadata)
	_, err = s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{})
	return err
}

func (s *Service) UpdateName(ctx context.Context, user views.User, name string) error {
	return s.UpdateNameWithMetadata(ctx, user, name, nil)
}

func (s *Service) UpdateNameWithMetadata(ctx context.Context, user views.User, name string, metadata CommandMetadata) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	userNameChangedID := uuidv7.NewString()
	event := NewUserNameChangedEvent(userNameChangedID, name, time.Now(), user.UserRegisteredID, metadata)
	if _, err := s.store.SaveEvents(ctx, []eventstore.DomainEvent{event}, eventstore.NoEventPosition, nil, eventstore.Query{}); err != nil {
		return err
	}
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserName(conn, dbsql.UpdateAuthUserNameParams{Name: name, Id: user.ID})
	})
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
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthAccountPassword(conn, dbsql.UpdateAuthAccountPasswordParams{
			Password: stringPtr(string(hash)),
			UserId:   userID,
		})
	})
}

func (s *Service) userByEmailWithPassword(ctx context.Context, emailAddress string) (views.User, string, error) {
	var row *dbsql.UserByEmailWithPasswordRes
	if err := s.db.ReadTX(ctx, func(conn *sqlite.Conn) error {
		var err error
		row, err = dbsql.OnceUserByEmailWithPassword(conn, emailAddress)
		return err
	}); err != nil {
		return views.User{}, "", err
	}
	if row == nil || row.Password == nil {
		return views.User{}, "", appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, *row.Password, nil
}

func userFromSessionRow(row *dbsql.UserBySessionTokenRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func userFromRegisteredRow(row *dbsql.UserByRegisteredIdRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func userFromIDOrRegisteredRow(row *dbsql.UserByIdorRegisteredIdRes) (views.User, error) {
	if row == nil {
		return views.User{}, appdb.ErrNoRows
	}
	return views.User{
		ID:               row.Id,
		UserRegisteredID: row.UserRegisteredId,
		Name:             row.Name,
		Username:         row.Username,
		Email:            row.Email,
		EmailVerified:    row.EmailVerified != 0,
		Image:            row.Image,
		Bio:              row.Bio,
		HeaderImageURL:   row.HeaderImageUrl,
	}, nil
}

func stringPtr(value string) *string {
	return &value
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
