package httpui

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

// Flush and Unwrap keep SSE streaming working: chi's Compress wrapper and
// http.ResponseController both need the wrapped writer to expose flushing,
// otherwise events stay in the response buffer and never reach the browser.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func requestLogging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			requestID := middleware.GetReqID(r.Context())
			metadata := map[string]any{
				"id":        requestID,
				"method":    r.Method,
				"path":      r.URL.Path,
				"datastar":  r.Header.Get("Datastar-Request") != "",
				"startedAt": started.UTC().Format(time.RFC3339Nano),
			}
			recorder := &statusRecorder{ResponseWriter: w}
			loggedRequest := r.WithContext(eventstore.WithRequestLogMetadata(r.Context(), metadata))
			next.ServeHTTP(recorder, loggedRequest)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			duration := time.Since(started)
			requestLog := eventstore.RequestLogMetadata(loggedRequest.Context())
			logger.Info("http request",
				"requestId", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"action", requestLog["action"],
				"actionFields", requestLog["actionFields"],
				"status", status,
				"bytes", recorder.bytes,
				"durationMs", duration.Milliseconds(),
				"datastar", metadata["datastar"],
			)
		})
	}
}
