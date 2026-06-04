package eventstore

import (
	"net"
	"net/http"
	"strings"
	"time"
)

type CommandMetadata map[string]any

func HTTPCommandMetadata(r *http.Request, userRegisteredID string) CommandMetadata {
	audit := map[string]any{
		"source":     "http",
		"capturedAt": time.Now().UTC().Format(time.RFC3339),
		"request": withoutNil(map[string]any{
			"id":             r.Header.Get("X-Request-Id"),
			"method":         r.Method,
			"host":           r.Host,
			"ipAddress":      clientIP(r),
			"forwardedFor":   r.Header.Get("X-Forwarded-For"),
			"userAgent":      r.UserAgent(),
			"referrer":       firstNonEmpty(r.Header.Get("Referer"), r.Header.Get("Referrer")),
			"origin":         r.Header.Get("Origin"),
			"acceptLanguage": r.Header.Get("Accept-Language"),
		}),
	}
	if userRegisteredID != "" {
		audit["actor"] = map[string]any{"userRegisteredId": userRegisteredID}
	}
	return CommandMetadata{"audit": audit}
}

func EventHandlerCommandMetadata(handlerName string, resolved ResolvedEvent) CommandMetadata {
	return CommandMetadata{"audit": map[string]any{
		"source":     "event-handler",
		"capturedAt": time.Now().UTC().Format(time.RFC3339),
		"eventHandler": map[string]any{
			"name": handlerName,
		},
		"reactedTo": map[string]any{
			"eventId":   resolved.Event.EventID,
			"eventType": resolved.Event.EventType,
			"position":  resolved.Position,
		},
	}}
}

func MergeMetadata(eventMetadata map[string]any, commandMetadata CommandMetadata) map[string]any {
	merged := map[string]any{}
	for key, value := range eventMetadata {
		merged[key] = value
	}
	for key, value := range commandMetadata {
		merged[key] = value
	}
	return merged
}

func withoutNil(values map[string]any) map[string]any {
	clean := map[string]any{}
	for key, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		clean[key] = value
	}
	return clean
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
