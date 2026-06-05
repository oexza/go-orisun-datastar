package httpui

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

func writeSSE(w http.ResponseWriter, r *http.Request, fn func(*datastar.ServerSentEventGenerator) error) {
	_ = fn(newSSE(w, r))
}

func newSSE(w http.ResponseWriter, r *http.Request) *datastar.ServerSentEventGenerator {
	return datastar.NewSSE(w, r, datastar.WithCompression())
}

func emptySSE(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
			return flashError(sse, err.Error())
		})
		return
	}
	writeSSE(w, r, clearFlash)
}

func clearNewTodoTitle(sse *datastar.ServerSentEventGenerator) error {
	return sse.MarshalAndPatchSignals(map[string]string{"flashMessage": "", "newTodoTitle": ""})
}

func alert(sse *datastar.ServerSentEventGenerator, message string) error {
	return flashError(sse, message)
}

func flashError(sse *datastar.ServerSentEventGenerator, message string) error {
	return sse.MarshalAndPatchSignals(map[string]string{"flashMessage": message})
}

func clearFlash(sse *datastar.ServerSentEventGenerator) error {
	return sse.MarshalAndPatchSignals(map[string]string{"flashMessage": ""})
}

func patchTempl(w http.ResponseWriter, r *http.Request, component templ.Component, opts ...datastar.PatchElementOption) {
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err := clearFlash(sse); err != nil {
			return err
		}
		return sse.PatchElementTempl(component, opts...)
	})
}
