package httpui

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
	"github.com/oexza/go-orisun-datastar/internal/viewstore"
)

func (s Server) monitoringRoutes(r chi.Router) {
	r.Get("/ops", s.opsDashboard)
	r.Get("/ops/stream", s.opsStream)
	r.Get("/ops/event-handlers", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eventstore.DefaultEventHandlerMetrics.Snapshot())
	})
}

func (s Server) opsDashboard(w http.ResponseWriter, r *http.Request) {
	setRequestAction(r, "ops.dashboard", nil)
	snapshot := opsSnapshot()
	_ = views.OpsDashboard(snapshot).Render(r.Context(), w)
}

func (s Server) opsStream(w http.ResponseWriter, r *http.Request) {
	setRequestAction(r, "ops.stream.connect", nil)
	sse := newSSE(w, r)
	key := viewstore.OpsMetricsKey()
	err := streamViewStoreFatMorph[views.OpsDashboardSnapshot](r.Context(), sse, viewStoreFatMorphConfig[views.OpsDashboardSnapshot]{
		Key:     key,
		Store:   s.ViewStore,
		Refresh: func(ctx context.Context) error { return viewstore.PutState(ctx, s.ViewStore, key, opsSnapshot()) },
		Subscribe: func(ctx context.Context, notify func()) (closeableSubscription, error) {
			subscription := newTickerSubscription(ctx, 2*time.Second, notify)
			return subscription, nil
		},
		Patch: func(sse *datastar.ServerSentEventGenerator, state views.OpsDashboardSnapshot) error {
			return sse.PatchElementTempl(views.OpsMetrics(state), datastar.WithSelector("#ops-metrics"))
		},
	})
	if err != nil {
		_ = alert(sse, err.Error())
	}
}

func opsSnapshot() views.OpsDashboardSnapshot {
	raw := eventstore.DefaultEventHandlerMetrics.Snapshot()
	snapshot := views.OpsDashboardSnapshot{Handlers: make([]views.OpsEventHandlerMetric, 0, len(raw.Handlers))}
	for _, handler := range raw.Handlers {
		snapshot.Handlers = append(snapshot.Handlers, views.OpsEventHandlerMetric{
			Name:                 handler.Name,
			Running:              handler.Running,
			LastEventID:          handler.LastEventID,
			LastEventType:        handler.LastEventType,
			LastCommitPosition:   handler.LastCommitPosition,
			LastCheckpointCommit: handler.LastCheckpointCommit,
			ProcessedCount:       handler.ProcessedCount,
			FailureCount:         handler.FailureCount,
			LastError:            handler.LastError,
			UpdatedAt:            handler.UpdatedAt.Format(time.RFC3339),
		})
	}
	return snapshot
}

type tickerSubscription struct {
	cancel context.CancelFunc
}

func newTickerSubscription(ctx context.Context, interval time.Duration, notify func()) *tickerSubscription {
	tickerCtx, cancel := context.WithCancel(ctx)
	subscription := &tickerSubscription{cancel: cancel}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		notify()
		for {
			select {
			case <-tickerCtx.Done():
				return
			case <-ticker.C:
				notify()
			}
		}
	}()
	return subscription
}

func (s *tickerSubscription) Close() error {
	s.cancel()
	return nil
}
