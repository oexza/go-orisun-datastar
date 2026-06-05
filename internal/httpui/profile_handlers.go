package httpui

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/example/go-orisun-datastar/internal/eventstore"
	"github.com/example/go-orisun-datastar/internal/views"
)

func (s Server) profileRoutes(r chi.Router) {
	r.Get("/profile", s.profilePage)
	r.Get("/profile/edit", s.profileEdit)
	r.Get("/profile/settings", s.settings)
	r.Post("/profile/bio", s.updateBio)
	r.Post("/profile/avatar", s.uploadAvatar)
	r.Post("/profile/header-image", s.uploadHeader)
	r.Post("/user/name", s.updateName)
	r.Post("/settings/change-password", s.changePassword)
}

func (s Server) profilePage(w http.ResponseWriter, r *http.Request) {
	_ = views.Profile(currentUser(r)).Render(r.Context(), w)
}

func (s Server) profileEdit(w http.ResponseWriter, r *http.Request) {
	_ = views.ProfileEdit(currentUser(r), nil).Render(r.Context(), w)
}

func (s Server) settings(w http.ResponseWriter, r *http.Request) {
	_ = views.Settings(currentUser(r)).Render(r.Context(), w)
}

func (s Server) updateBio(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	err := s.Profile.UpdateBioWithMetadata(r.Context(), user, r.FormValue("bio"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID))
	if err != nil {
		_ = views.ProfileEdit(user, map[string]string{"bio": err.Error()}).Render(r.Context(), w)
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/profile") })
}

func (s Server) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	s.uploadImage(w, r, false)
}

func (s Server) uploadHeader(w http.ResponseWriter, r *http.Request) {
	s.uploadImage(w, r, true)
}

func (s Server) uploadImage(w http.ResponseWriter, r *http.Request, header bool) {
	user := currentUser(r)
	data, contentType, err := readUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := s.Profile.UploadImageWithMetadata(r.Context(), user, data, contentType, header, eventstore.HTTPCommandMetadata(r, user.UserRegisteredID)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (s Server) updateName(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	err := s.Auth.UpdateNameWithMetadata(r.Context(), user, r.FormValue("name"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID))
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err != nil {
			return alert(sse, err.Error())
		}
		return sse.Redirect("/profile/settings")
	})
}

func (s Server) changePassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	err := s.Auth.ChangePasswordWithMetadata(r.Context(), user, r.FormValue("currentPassword"), r.FormValue("newPassword"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID))
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err != nil {
			return alert(sse, err.Error())
		}
		return sse.Redirect("/profile/settings")
	})
}

func readUpload(r *http.Request) ([]byte, string, error) {
	if err := r.ParseMultipartForm(6 * 1024 * 1024); err != nil {
		return nil, "", err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 6*1024*1024))
	if err != nil {
		return nil, "", err
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType, nil
}
