package httpui

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
)

func setupReload(r chi.Router) {
	reloadChan := make(chan struct{}, 1)
	var initialReload sync.Once

	r.Get("/reload", func(w http.ResponseWriter, r *http.Request) {
		sse := newSSE(w, r)
		reload := func() { _ = sse.ExecuteScript("window.location.reload()") }
		initialReload.Do(reload)
		select {
		case <-reloadChan:
			reload()
		case <-r.Context().Done():
		}
	})

	r.Get("/hotreload", func(w http.ResponseWriter, r *http.Request) {
		select {
		case reloadChan <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
