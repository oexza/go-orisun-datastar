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
	if len(input.Password) < 6 {
		return views.User{}, errors.New("invalid registration input")
	}

	registered, err := RegisterUserCommandHandler(ctx, RegisterUserCommand{
		Username:    input.Username,
		Email:       input.Email,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		YearOfBirth: input.YearOfBirth,
		Metadata:    metadata,
	}, s.store, s.retriever)
	if err != nil {
		return views.User{}, err
	}

	userID := uuidv7.NewString()
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return views.User{}, err
	}

	name := strings.TrimSpace(registered.FirstName + " " + registered.LastName)
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		if err := dbsql.OnceCreateAuthUser(conn, dbsql.CreateAuthUserParams{
			Id:               userID,
			Name:             name,
			Email:            registered.Email,
			Username:         stringPtr(registered.Username),
			UserRegisteredId: registered.UserRegisteredID,
		}); err != nil {
			return err
		}
		return dbsql.OnceCreateAuthAccount(conn, dbsql.CreateAuthAccountParams{
			Id:        uuidv7.NewString(),
			AccountId: registered.Email,
			UserId:    userID,
			Password:  stringPtr(string(hash)),
		})
	}); err != nil {
		return views.User{}, err
	}

	user := views.User{ID: userID, UserRegisteredID: registered.UserRegisteredID, Name: name, Username: registered.Username, Email: registered.Email}
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
	result, err := GenerateEmailVerificationOTPCommandHandler(ctx, GenerateEmailVerificationOTPCommand{
		User:     user,
		Metadata: metadata,
	}, s.store, s.retriever)
	if err != nil {
		return err
	}
	if result.Skipped {
		return nil
	}
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         result.EmailVerificationOTPGeneratedID,
			Identifier: "email:" + user.UserRegisteredID,
			Value:      result.Code,
			ExpiresAt:  appdb.SQLTime(result.ExpiresAt),
		})
	}); err != nil {
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
	return ValidateEmailVerificationOTPCommandHandler(ctx, ValidateEmailVerificationOTPCommand{
		User:     user,
		Code:     code,
		Metadata: metadata,
	}, s.store, s.retriever)
}

func (s *Service) RequestPasswordReset(ctx context.Context, emailAddress string) error {
	return s.RequestPasswordResetWithMetadata(ctx, emailAddress, nil)
}

func (s *Service) RequestPasswordResetWithMetadata(ctx context.Context, emailAddress string, metadata CommandMetadata) error {
	user, _, err := s.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
	if err != nil {
		return nil
	}
	result, err := RequestPasswordResetCommandHandler(ctx, RequestPasswordResetCommand{
		User:     user,
		Metadata: metadata,
	}, s.store, s.retriever)
	if err != nil {
		return err
	}
	if err := s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceCreateAuthVerification(conn, dbsql.CreateAuthVerificationParams{
			Id:         result.PasswordResetRequestedID,
			Identifier: "password-reset:" + user.ID,
			Value:      result.Token,
			ExpiresAt:  appdb.SQLTime(result.ExpiresAt),
		})
	}); err != nil {
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
	return ResetPasswordCommandHandler(ctx, ResetPasswordCommand{
		User:                     user,
		PasswordResetRequestedID: verification.Id,
		Metadata:                 metadata,
	}, s.store, s.retriever)
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
	return ChangePasswordCommandHandler(ctx, ChangePasswordCommand{
		User:     user,
		Metadata: metadata,
	}, s.store, s.retriever)
}

func (s *Service) UpdateName(ctx context.Context, user views.User, name string) error {
	return s.UpdateNameWithMetadata(ctx, user, name, nil)
}

func (s *Service) UpdateNameWithMetadata(ctx context.Context, user views.User, name string, metadata CommandMetadata) error {
	result, err := UpdateUserNameCommandHandler(ctx, UpdateUserNameCommand{
		User:     user,
		Name:     name,
		Metadata: metadata,
	}, s.store, s.retriever)
	if err != nil {
		return err
	}
	if result.Skipped {
		return nil
	}
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceUpdateAuthUserName(conn, dbsql.UpdateAuthUserNameParams{Name: result.Name, Id: user.ID})
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
