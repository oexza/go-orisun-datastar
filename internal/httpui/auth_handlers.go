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
	registered, err := auth.RegisterUserCommandHandler(r.Context(), auth.RegisterUserCommand{
		Username: r.FormValue("username"), Email: r.FormValue("email"), Password: r.FormValue("password"),
		FirstName: r.FormValue("firstName"), LastName: r.FormValue("lastName"), YearOfBirth: year,
		Metadata: eventstore.HTTPCommandMetadata(r, ""),
	}, s.EventSaver, s.EventRetriever)
	if err != nil {
		patchTempl(w, r, views.RegisterForm(map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		return sse.Redirect("/register/" + registered.UserRegisteredID + "/validate-email")
	})
}

func (s Server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return flashError(sse, err.Error()) })
		return
	}
	user, token, err := s.Sessions.Login(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		patchTempl(w, r, views.LoginForm(map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	s.Sessions.SetSessionCookie(w, token)
	path := "/todos"
	if !user.EmailVerified {
		path = "/register/" + user.ID + "/validate-email"
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect(path) })
}

func (s Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(s.Sessions.SessionCookieName()); err == nil {
		_ = s.Sessions.Logout(r.Context(), cookie.Value)
	}
	s.Sessions.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	_, _ = auth.RequestPasswordResetCommandHandler(r.Context(), auth.RequestPasswordResetCommand{
		EmailAddress: r.FormValue("email"),
		Metadata:     eventstore.HTTPCommandMetadata(r, ""),
	}, s.PasswordCredentials, s.EventSaver, s.EventRetriever)
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	token := chi.URLParam(r, "token")
	if err := auth.ResetPasswordCommandHandler(r.Context(), auth.ResetPasswordCommand{
		Token:    token,
		Password: r.FormValue("password"),
		Metadata: eventstore.HTTPCommandMetadata(r, ""),
	}, s.Verifications, s.AuthUsers, s.EventSaver, s.EventRetriever); err != nil {
		patchTempl(w, r, views.ResetPasswordForm(token, map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) validateEmail(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	userID := chi.URLParam(r, "userID")
	if err := auth.ValidateEmailVerificationOTPForUserCommandHandler(r.Context(), userID, r.FormValue("otp"), eventstore.HTTPCommandMetadata(r, ""), s.AuthUsers, s.EventSaver, s.EventRetriever); err != nil {
		patchTempl(w, r, views.ValidateEmailForm(userID, map[string]string{"error": err.Error()}), datastar.WithSelectorID("auth-page"))
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/login") })
}

func (s Server) sendOTP(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	user, err := s.AuthUsers.UserByIDOrRegisteredID(r.Context(), userID)
	if err == nil {
		_, _ = auth.GenerateEmailVerificationOTPCommandHandler(r.Context(), auth.GenerateEmailVerificationOTPCommand{
			User:     user,
			Metadata: eventstore.HTTPCommandMetadata(r, user.UserRegisteredID),
		}, s.EventSaver, s.EventRetriever)
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		return sse.Redirect("/register/" + userID + "/validate-email")
	})
}
