package httpui

import (
	"net/http"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
)

func setRequestAction(r *http.Request, action string, fields map[string]any) {
	eventstore.SetRequestAction(r.Context(), action, fields)
}
