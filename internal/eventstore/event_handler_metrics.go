package eventstore

import (
	"sync"
	"time"
)

type EventHandlerMetric struct {
	Name                  string    `json:"name"`
	Running               bool      `json:"running"`
	LastEventID           string    `json:"lastEventId,omitempty"`
	LastEventType         string    `json:"lastEventType,omitempty"`
	LastCommitPosition    int64     `json:"lastCommitPosition"`
	LastPreparePosition   int64     `json:"lastPreparePosition"`
	LastCheckpointCommit  int64     `json:"lastCheckpointCommit"`
	LastCheckpointPrepare int64     `json:"lastCheckpointPrepare"`
	ProcessedCount        int64     `json:"processedCount"`
	FailureCount          int64     `json:"failureCount"`
	LastError             string    `json:"lastError,omitempty"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type EventHandlerMetricsSnapshot struct {
	Handlers []EventHandlerMetric `json:"handlers"`
}

type EventHandlerMetricsSink interface {
	HandlerStarted(name string, position Position)
	HandlerStopped(name string)
	EventProcessed(name string, event ResolvedEvent)
	EventFailed(name string, event ResolvedEvent, err error)
	CheckpointUpdated(name string, position Position)
	Snapshot() EventHandlerMetricsSnapshot
}

type MemoryEventHandlerMetrics struct {
	mu       sync.Mutex
	handlers map[string]EventHandlerMetric
}

var DefaultEventHandlerMetrics EventHandlerMetricsSink = NewMemoryEventHandlerMetrics()

func NewMemoryEventHandlerMetrics() *MemoryEventHandlerMetrics {
	return &MemoryEventHandlerMetrics{handlers: map[string]EventHandlerMetric{}}
}

func (m *MemoryEventHandlerMetrics) HandlerStarted(name string, position Position) {
	m.update(name, func(metric *EventHandlerMetric) {
		metric.Running = true
		metric.LastCheckpointCommit = position.Commit
		metric.LastCheckpointPrepare = position.Prepare
	})
}

func (m *MemoryEventHandlerMetrics) HandlerStopped(name string) {
	m.update(name, func(metric *EventHandlerMetric) {
		metric.Running = false
	})
}

func (m *MemoryEventHandlerMetrics) EventProcessed(name string, event ResolvedEvent) {
	m.update(name, func(metric *EventHandlerMetric) {
		metric.LastEventID = event.Event.EventID
		metric.LastEventType = event.Event.EventType
		metric.LastCommitPosition = event.Position.Commit
		metric.LastPreparePosition = event.Position.Prepare
		metric.ProcessedCount++
		metric.LastError = ""
	})
}

func (m *MemoryEventHandlerMetrics) EventFailed(name string, event ResolvedEvent, err error) {
	m.update(name, func(metric *EventHandlerMetric) {
		metric.LastEventID = event.Event.EventID
		metric.LastEventType = event.Event.EventType
		metric.LastCommitPosition = event.Position.Commit
		metric.LastPreparePosition = event.Position.Prepare
		metric.FailureCount++
		metric.LastError = err.Error()
	})
}

func (m *MemoryEventHandlerMetrics) CheckpointUpdated(name string, position Position) {
	m.update(name, func(metric *EventHandlerMetric) {
		metric.LastCheckpointCommit = position.Commit
		metric.LastCheckpointPrepare = position.Prepare
	})
}

func (m *MemoryEventHandlerMetrics) Snapshot() EventHandlerMetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	handlers := make([]EventHandlerMetric, 0, len(m.handlers))
	for _, metric := range m.handlers {
		handlers = append(handlers, metric)
	}
	return EventHandlerMetricsSnapshot{Handlers: handlers}
}

func (m *MemoryEventHandlerMetrics) update(name string, apply func(*EventHandlerMetric)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	metric := m.handlers[name]
	metric.Name = name
	apply(&metric)
	metric.UpdatedAt = time.Now().UTC()
	m.handlers[name] = metric
}

type noopEventHandlerMetrics struct{}

func (noopEventHandlerMetrics) HandlerStarted(string, Position)          {}
func (noopEventHandlerMetrics) HandlerStopped(string)                    {}
func (noopEventHandlerMetrics) EventProcessed(string, ResolvedEvent)     {}
func (noopEventHandlerMetrics) EventFailed(string, ResolvedEvent, error) {}
func (noopEventHandlerMetrics) CheckpointUpdated(string, Position)       {}
func (noopEventHandlerMetrics) Snapshot() EventHandlerMetricsSnapshot {
	return EventHandlerMetricsSnapshot{}
}
