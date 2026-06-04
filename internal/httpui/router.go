package httpui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/example/hono-event-starter-go/internal/auth"
	"github.com/example/hono-event-starter-go/internal/eventstore"
	"github.com/example/hono-event-starter-go/internal/features/profile"
	"github.com/example/hono-event-starter-go/internal/features/todo"
	"github.com/example/hono-event-starter-go/internal/resources"
	"github.com/example/hono-event-starter-go/internal/views"
	"github.com/example/hono-event-starter-go/internal/viewstore"
)

type contextKey string

const userKey contextKey = "user"

type Server struct {
	Auth        *auth.Service
	Todos       *todo.Service
	Profile     *profile.Service
	Subscriber  eventstore.MessageSubscriber
	ViewStore   viewstore.Store
	Development bool
}

func (s Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	})
	if s.Development {
		setupReload(r)
	}
	r.Handle("/static/*", resources.Handler())
	r.Get("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/todos", http.StatusFound) })

	s.authRoutes(r)

	r.Group(func(r chi.Router) {
		r.Use(s.requireVerifiedEmail)
		s.todoRoutes(r)
		s.profileRoutes(r)
	})

	return r
}

func render(component interface {
	Render(context.Context, io.Writer) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = component.Render(r.Context(), w)
	}
}

func (s Server) requireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok, err := s.Auth.CurrentUser(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !user.EmailVerified {
			http.Redirect(w, r, "/register/"+user.ID+"/validate-email", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, user)))
	})
}

func currentUser(r *http.Request) views.User {
	user, _ := r.Context().Value(userKey).(views.User)
	return user
}

func (s Server) sessionID(r *http.Request) string {
	cookie, err := r.Cookie(s.Auth.SessionCookieName())
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return currentUser(r).UserRegisteredID
}
