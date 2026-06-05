package httpui

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/auth"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

func (s Server) authRoutes(r chi.Router) {
	r.Get("/register", render(views.Register(nil)))
	r.Post("/register", s.register)
	r.Get("/login", render(views.Login(nil)))
	r.Post("/login", s.login)
	r.Post("/logout", s.logout)
	r.Get("/forgot-password", render(views.ForgotPassword(nil)))
	r.Post("/forgot-password", s.forgotPassword)
	r.Get("/reset-password/{token}", func(w http.ResponseWriter, r *http.Request) {
		_ = views.ResetPassword(chi.URLParam(r, "token"), nil).Render(r.Context(), w)
	})
	r.Post("/reset-password/{token}", s.resetPassword)
	r.Get("/register/{userID}/validate-email", func(w http.ResponseWriter, r *http.Request) {
		_ = views.ValidateEmail(chi.URLParam(r, "userID"), nil).Render(r.Context(), w)
	})
	r.Post("/register/{userID}/validate-email", s.validateEmail)
	r.Post("/register/{userID}/send-email-validation-otp", s.sendOTP)
}

func (s Server) register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return flashError(sse, err.Error()) })
		return
	}
	year, _ := strconv.Atoi(r.FormValue("yearOfBirth"))
	user, err := s.Auth.Register(r.Context(), auth.RegisterInput{
		Username: r.FormValue("username"), Email: r.FormValue("email"), Password: r.FormValue("password"),
		FirstName: r.FormValue("firstName"), LastName: r.FormValue("lastName"), YearOfBirth: year,
		Metadata: eventstore.HTTPCommandMetadata(r, ""),
	})
	if err != nil {
		patchTempl(w, r, views.RegisterForm(map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		return sse.Redirect("/register/" + user.ID + "/validate-email")
	})
}

func (s Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return flashError(sse, err.Error()) })
		return
	}
	user, token, err := s.Auth.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		patchTempl(w, r, views.LoginForm(map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	s.Auth.SetSessionCookie(w, token)
	path := "/todos"
	if !user.EmailVerified {
		path = "/register/" + user.ID + "/validate-email"
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect(path) })
}

func (s Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(s.Auth.SessionCookieName()); err == nil {
		_ = s.Auth.Logout(r.Context(), cookie.Value)
	}
	s.Auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	_ = s.Auth.RequestPasswordResetWithMetadata(r.Context(), r.FormValue("email"), eventstore.HTTPCommandMetadata(r, ""))
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token := chi.URLParam(r, "token")
	if err := s.Auth.ResetPasswordWithMetadata(r.Context(), token, r.FormValue("password"), eventstore.HTTPCommandMetadata(r, "")); err != nil {
		patchTempl(w, r, views.ResetPasswordForm(token, map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) validateEmail(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := chi.URLParam(r, "userID")
	if err := s.Auth.ValidateOTPWithMetadata(r.Context(), userID, r.FormValue("otp"), eventstore.HTTPCommandMetadata(r, "")); err != nil {
		patchTempl(w, r, views.ValidateEmailForm(userID, map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) sendOTP(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	user, err := s.Auth.UserByIDOrRegisteredID(r.Context(), userID)
	if err == nil {
		_ = s.Auth.GenerateEmailVerificationOTPWithMetadata(r.Context(), user, eventstore.HTTPCommandMetadata(r, user.UserRegisteredID))
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		return sse.Redirect("/register/" + userID + "/validate-email")
	})
}
