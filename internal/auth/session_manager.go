package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"golang.org/x/crypto/bcrypt"
	"zombiezen.com/go/sqlite"
)

type SessionManager struct {
	db            *appdb.DB
	users         *AuthUserStore
	secureCookie  bool
	sessionCookie string
}

func NewSessionManager(db *appdb.DB, users *AuthUserStore, secureCookie bool) *SessionManager {
	return &SessionManager{
		db:            db,
		users:         users,
		secureCookie:  secureCookie,
		sessionCookie: "go-event-starter-session",
	}
}

func (s *SessionManager) Login(ctx context.Context, emailAddress, password string) (views.User, string, error) {
	user, hash, err := s.users.userByEmailWithPassword(ctx, strings.ToLower(strings.TrimSpace(emailAddress)))
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

func (s *SessionManager) Logout(ctx context.Context, token string) error {
	return s.db.WriteTX(ctx, func(conn *sqlite.Conn) error {
		return dbsql.OnceDeleteAuthSessionByToken(conn, token)
	})
}

func (s *SessionManager) CurrentUser(ctx context.Context, r *http.Request) (views.User, bool, error) {
	cookie, err := r.Cookie(s.sessionCookie)
	if err != nil || cookie.Value == "" {
		return views.User{}, false, nil
	}
	user, err := s.users.UserBySessionToken(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, appdb.ErrNoRows) {
			return views.User{}, false, nil
		}
		return views.User{}, false, err
	}
	return user, true, nil
}

func (s *SessionManager) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie, Expires: time.Now().Add(90 * 24 * time.Hour)})
}

func (s *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: s.sessionCookie, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secureCookie, MaxAge: -1})
}

func (s *SessionManager) SessionCookieName() string {
	return s.sessionCookie
}
