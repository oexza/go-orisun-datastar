package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"

	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"golang.org/x/crypto/bcrypt"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"zombiezen.com/go/sqlite"
)

type AccountCommands struct {
	db        *appdb.DB
	users     *AuthUserStore
	store     eventstore.Saver
	retriever eventstore.Retriever
}

func NewAccountCommands(db *appdb.DB, users *AuthUserStore, saver eventstore.Saver, retriever eventstore.Retriever) *AccountCommands {
	return &AccountCommands{db: db, users: users, store: saver, retriever: retriever}
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

func (s *AccountCommands) Register(ctx context.Context, input RegisterInput) (views.User, error) {
	return s.RegisterWithMetadata(ctx, input, input.Metadata)
}

func (s *AccountCommands) RegisterWithMetadata(ctx context.Context, input RegisterInput, metadata CommandMetadata) (views.User, error) {
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

func (s *AccountCommands) GenerateEmailVerificationOTP(ctx context.Context, user views.User) error {
	return s.generateEmailVerificationOTP(ctx, user, nil)
}

func (s *AccountCommands) GenerateEmailVerificationOTPWithMetadata(ctx context.Context, user views.User, metadata CommandMetadata) error {
	return s.generateEmailVerificationOTP(ctx, user, metadata)
}

func (s *AccountCommands) generateEmailVerificationOTP(ctx context.Context, user views.User, metadata CommandMetadata) error {
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

func (s *AccountCommands) ValidateOTP(ctx context.Context, userID, code string) error {
	return s.ValidateOTPWithMetadata(ctx, userID, code, nil)
}

func (s *AccountCommands) ValidateOTPWithMetadata(ctx context.Context, userID, code string, metadata CommandMetadata) error {
	user, err := s.users.UserByIDOrRegisteredID(ctx, userID)
	if err != nil {
		return err
	}
	return ValidateEmailVerificationOTPCommandHandler(ctx, ValidateEmailVerificationOTPCommand{
		User:     user,
		Code:     code,
		Metadata: metadata,
	}, s.store, s.retriever)
}

func (s *AccountCommands) RequestPasswordReset(ctx context.Context, emailAddress string) error {
	return s.RequestPasswordResetWithMetadata(ctx, emailAddress, nil)
}

func (s *AccountCommands) RequestPasswordResetWithMetadata(ctx context.Context, emailAddress string, metadata CommandMetadata) error {
	user, _, err := s.users.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
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

func (s *AccountCommands) ResetPassword(ctx context.Context, token, password string) error {
	return s.ResetPasswordWithMetadata(ctx, token, password, nil)
}

func (s *AccountCommands) ResetPasswordWithMetadata(ctx context.Context, token, password string, metadata CommandMetadata) error {
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
	user, err := s.users.UserByIDOrRegisteredID(ctx, userID)
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

func (s *AccountCommands) ChangePassword(ctx context.Context, user views.User, currentPassword, newPassword string) error {
	return s.ChangePasswordWithMetadata(ctx, user, currentPassword, newPassword, nil)
}

func (s *AccountCommands) ChangePasswordWithMetadata(ctx context.Context, user views.User, currentPassword, newPassword string, metadata CommandMetadata) error {
	_, hash, err := s.users.userByEmailWithPassword(ctx, user.Email)
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

func (s *AccountCommands) UpdateName(ctx context.Context, user views.User, name string) error {
	return s.UpdateNameWithMetadata(ctx, user, name, nil)
}

func (s *AccountCommands) UpdateNameWithMetadata(ctx context.Context, user views.User, name string, metadata CommandMetadata) error {
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

func (s *AccountCommands) setPassword(ctx context.Context, userID, password string) error {
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
