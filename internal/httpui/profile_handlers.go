package httpui

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/appdb"
	"github.com/oexza/go-orisun-datastar/internal/auth"
	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/features/profile"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"github.com/oexza/go-orisun-datastar/internal/viewstore"
)

type profileViewState struct {
	User views.User `json:"user"`
}

func (s Server) profileRoutes(r chi.Router) {
	r.Get("/profile", s.profilePage)
	r.Get("/profile/stream", s.profileStream)
	r.Get("/profile/edit", s.profileEdit)
	r.Get("/profile/settings", s.settings)
	r.Post("/profile/bio", s.updateBio)
	r.Post("/profile/avatar", s.uploadAvatar)
	r.Post("/profile/header-image", s.uploadHeader)
	r.Post("/user/name", s.updateName)
	r.Post("/settings/change-password", s.changePassword)
}

func (s Server) profilePage(w http.ResponseWriter, r *http.Request) {
	user, err := s.profileUser(r.Context(), currentUser(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = views.Profile(user).Render(r.Context(), w)
}

func (s Server) profileStream(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	sse := newSSE(w, r)
	ctx := r.Context()
	key := viewstore.ProfileKey(s.sessionID(r), user.UserRegisteredID)

	updates := make(chan struct{}, 1)
	if err := s.refreshProfileViewState(ctx, key, user); err != nil {
		_ = alert(sse, err.Error())
		return
	}

	watcher, err := s.ViewStore.Watch(ctx, key, viewstore.WatchOptions{IgnoreDeletes: true})
	if err != nil {
		_ = alert(sse, err.Error())
		return
	}
	defer watcher.Stop()

	sub, err := s.Subscriber.Subscribe(ctx, profile.Channel(user.UserRegisteredID), func(context.Context, []byte) {
		notifyOnce(updates)
	})
	if err != nil {
		_ = alert(sse, err.Error())
		return
	}
	defer sub.Close()

	notifyOnce(updates)
	for {
		select {
		case <-ctx.Done():
			return
		case <-updates:
			if err := s.refreshProfileViewState(ctx, key, user); err != nil {
				_ = alert(sse, err.Error())
				return
			}
		case entry, ok := <-watcher.Updates():
			if !ok {
				return
			}
			var state profileViewState
			if err := entry.JSON(&state); err != nil {
				_ = alert(sse, err.Error())
				return
			}
			if err := sse.PatchElementTempl(views.ProfilePanel(state.User), datastar.WithSelector("#profile-panel"), datastar.WithMode(datastar.ElementPatchModeInner)); err != nil {
				return
			}
		}
	}
}

func (s Server) profileEdit(w http.ResponseWriter, r *http.Request) {
	user, err := s.profileUser(r.Context(), currentUser(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = views.ProfileEdit(user, nil).Render(r.Context(), w)
}

func (s Server) settings(w http.ResponseWriter, r *http.Request) {
	_ = views.Settings(currentUser(r)).Render(r.Context(), w)
}

func (s Server) updateBio(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	err := profile.UpdateProfileBioCommandHandler(r.Context(), profile.UpdateProfileBioCommand{
		User:     user,
		Bio:      r.FormValue("bio"),
		Metadata: eventstore.HTTPCommandMetadata(r, user.UserRegisteredID),
	}, s.EventSaver, s.EventRetriever)
	if err != nil {
		patchTempl(w, r, views.ProfileEditPanel(user, map[string]string{"bio": err.Error()}), datastar.WithSelectorID("profile-edit-page"))
		return
	}
	actionSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return sse.Redirect("/profile") })
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
	if _, err := profile.UploadProfileImageCommandHandler(r.Context(), profile.UploadProfileImageCommand{
		User:        user,
		Data:        data,
		ContentType: contentType,
		Header:      header,
		Metadata:    eventstore.HTTPCommandMetadata(r, user.UserRegisteredID),
	}, s.EventSaver, s.EventRetriever, s.ProfileStorage); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (s Server) updateName(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	_, err := auth.UpdateUserNameCommandHandler(r.Context(), auth.UpdateUserNameCommand{
		User:     user,
		Name:     r.FormValue("name"),
		Metadata: eventstore.HTTPCommandMetadata(r, user.UserRegisteredID),
	}, s.EventSaver, s.EventRetriever)
	actionSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err != nil {
			return alert(sse, err.Error())
		}
		return sse.Redirect("/profile/settings")
	})
}

func (s Server) changePassword(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	err := auth.ChangePasswordCommandHandler(r.Context(), auth.ChangePasswordCommand{
		User:            user,
		CurrentPassword: r.FormValue("currentPassword"),
		NewPassword:     r.FormValue("newPassword"),
		Metadata:        eventstore.HTTPCommandMetadata(r, user.UserRegisteredID),
	}, s.PasswordCredentials, s.EventSaver, s.EventRetriever)
	actionSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err != nil {
			return alert(sse, err.Error())
		}
		return sse.Redirect("/profile/settings")
	})
}

func (s Server) refreshProfileViewState(ctx context.Context, key string, current views.User) error {
	user, err := s.profileUser(ctx, current)
	if err != nil {
		return err
	}
	return viewstore.PutState(ctx, s.ViewStore, key, profileViewState{User: user})
}

func (s Server) profileUser(ctx context.Context, current views.User) (views.User, error) {
	user, err := s.Profiles.User(ctx, current.UserRegisteredID)
	if err != nil {
		if errors.Is(err, appdb.ErrNoRows) {
			return current, nil
		}
		return views.User{}, err
	}
	user.EmailVerified = current.EmailVerified
	return user, nil
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
