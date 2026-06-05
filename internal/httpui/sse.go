package httpui

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

func writeSSE(w http.ResponseWriter, r *http.Request, fn func(*datastar.ServerSentEventGenerator) error) {
	_ = fn(datastar.NewSSE(w, r))
}

func emptySSE(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, eventstore.ErrNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeSSE(w, r, func(*datastar.ServerSentEventGenerator) error { return nil })
}

func clearInput(sse *datastar.ServerSentEventGenerator, id string) error {
	return sse.ExecuteScript(`{ const el = document.getElementById(` + strconv.Quote(id) + `); if (el) { el.value = ""; el.style.height = "auto"; } }`)
}

func alert(sse *datastar.ServerSentEventGenerator, message string) error {
	return sse.ExecuteScript(`alert(` + strconv.Quote(message) + `)`)
}
