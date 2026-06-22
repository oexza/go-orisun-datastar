package httpui

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/viewstore"
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

func actionSSE(w http.ResponseWriter, r *http.Request, action func(*datastar.ServerSentEventGenerator) error) {
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error {
		if err := clearFlash(sse); err != nil {
			return err
		}
		return action(sse)
	})
}

func notifyOnce(updates chan<- struct{}) {
	select {
	case updates <- struct{}{}:
	default:
	}
}

func watchViewState[T any](ctx context.Context, watcher viewstore.Watcher, onUpdate func(T) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case entry, ok := <-watcher.Updates():
			if !ok {
				return nil
			}
			var state T
			if err := entry.JSON(&state); err != nil {
				return err
			}
			if err := onUpdate(state); err != nil {
				return err
			}
		}
	}
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
