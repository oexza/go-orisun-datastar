package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oexza/go-orisun-datastar/internal/dbsql"
	"github.com/oexza/go-orisun-datastar/internal/uuidv7"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"golang.org/x/crypto/bcrypt"
)

type SessionManager struct {
	queries       *dbsql.Queries
	users         *AuthUserStore
	secureCookie  bool
	sessionCookie string
}

func NewSessionManager(db *pgxpool.Pool, users *AuthUserStore, secureCookie bool) *SessionManager {
	return &SessionManager{
		queries:       dbsql.New(db),
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
	err = s.queries.CreateAuthSession(ctx, dbsql.CreateAuthSessionParams{
		ID:        uuidv7.NewString(),
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: pgTime(time.Now().Add(90 * 24 * time.Hour)),
	})
	return user, token, err
}

func (s *SessionManager) Logout(ctx context.Context, token string) error {
	return s.queries.DeleteAuthSessionByToken(ctx, token)
}

func (s *SessionManager) CurrentUser(ctx context.Context, r *http.Request) (views.User, bool, error) {
	cookie, err := r.Cookie(s.sessionCookie)
	if err != nil || cookie.Value == "" {
		return views.User{}, false, nil
	}
	user, err := s.users.UserBySessionToken(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
