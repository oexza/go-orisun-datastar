package eventstore

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"
)

type HandleResolvedEvent func(context.Context, ResolvedEvent) error

type GlobalEventHandler struct {
	subscriber      Subscriber
	checkpointer    Checkpointer
	name            string
	query           Query
	logger          *slog.Logger
	handleEvent     HandleResolvedEvent
	maxEventRetries int

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type GlobalEventHandlerConfig struct {
	Subscriber      Subscriber
	Checkpointer    Checkpointer
	Name            string
	Query           Query
	Logger          *slog.Logger
	MaxEventRetries int
	HandleEvent     HandleResolvedEvent
}

func NewGlobalEventHandler(config GlobalEventHandlerConfig) (*GlobalEventHandler, error) {
	if config.Subscriber == nil {
		return nil, errors.New("events subscriber is required")
	}
	if config.Checkpointer == nil {
		return nil, errors.New("event handler checkpointer is required")
	}
	if config.Name == "" {
		return nil, errors.New("event handler name is required")
	}
	if config.HandleEvent == nil {
		return nil, errors.New("event handler callback is required")
	}
	if config.MaxEventRetries < -1 {
		return nil, errors.New("max event retries must be -1 or a positive integer")
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.MaxEventRetries == 0 {
		config.MaxEventRetries = -1
	}
	return &GlobalEventHandler{
		subscriber:      config.Subscriber,
		checkpointer:    config.Checkpointer,
		name:            config.Name,
		query:           config.Query,
		logger:          config.Logger,
		handleEvent:     config.HandleEvent,
		maxEventRetries: config.MaxEventRetries,
	}, nil
}

func (h *GlobalEventHandler) StartSubscribing(ctx context.Context) error {
	h.mu.Lock()
	if h.cancel != nil {
		h.mu.Unlock()
		h.logger.Info("event handler is already subscribing", "handler", h.name)
		return nil
	}
	subscriptionCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.mu.Unlock()

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.run(subscriptionCtx)
	}()
	return nil
}

func (h *GlobalEventHandler) StopSubscribing() {
	h.mu.Lock()
	cancel := h.cancel
	h.cancel = nil
	h.mu.Unlock()
	if cancel != nil {
		h.logger.Info("stopping event handler subscription", "handler", h.name)
		cancel()
		h.wg.Wait()
	}
}

func (h *GlobalEventHandler) run(ctx context.Context) {
	for attempt := 0; ; attempt++ {
		position, ok, err := h.checkpointer.GetCheckpoint(ctx, h.name)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.logger.Error("event handler checkpoint lookup failed", "handler", h.name, "attempt", attempt+1, "err", err)
			if waitErr := sleepWithContext(ctx, retryDelay(5*time.Second, 60*time.Second, 2*time.Second, attempt)); waitErr != nil {
				return
			}
			continue
		}
		if !ok {
			position = NoEventPosition
		}

		h.logger.Info("starting event handler subscription", "handler", h.name, "commit", position.Commit, "prepare", position.Prepare, "attempt", attempt+1)
		err = h.subscriber.SubscribeToEvents(ctx, h.name, position, h.query, func(ctx context.Context, event ResolvedEvent) error {
			if err := h.retryEventProcessing(ctx, event, 0); err != nil {
				return err
			}
			return h.retryCheckpointUpdate(ctx, event.Position, 0)
		})

		if ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return
		}

		if err != nil {
			h.logger.Error("event handler subscription ended", "handler", h.name, "attempt", attempt+1, "err", err)
		} else {
			h.logger.Warn("event handler subscription ended without error", "handler", h.name, "attempt", attempt+1)
		}
		if err := sleepWithContext(ctx, retryDelay(5*time.Second, 60*time.Second, 2*time.Second, attempt)); err != nil {
			return
		}
	}
}

func (h *GlobalEventHandler) retryEventProcessing(ctx context.Context, event ResolvedEvent, retryCount int) error {
	for {
		err := h.handleEvent(ctx, event)
		if err == nil {
			h.logger.Info("event handler processed event", "handler", h.name, "eventId", event.Event.EventID, "eventType", event.Event.EventType, "retryCount", retryCount)
			return nil
		}

		h.logger.Error("event handler failed processing event", "handler", h.name, "eventId", event.Event.EventID, "eventType", event.Event.EventType, "retryCount", retryCount, "err", err)
		if h.maxEventRetries != -1 && retryCount >= h.maxEventRetries-1 {
			return err
		}
		if waitErr := sleepWithContext(ctx, retryDelay(time.Second, 30*time.Second, time.Second, retryCount)); waitErr != nil {
			return waitErr
		}
		retryCount++
	}
}

func (h *GlobalEventHandler) retryCheckpointUpdate(ctx context.Context, position Position, retryCount int) error {
	for {
		err := h.checkpointer.UpdateCheckpoint(ctx, h.name, position)
		if err == nil {
			h.logger.Info("event handler checkpoint updated", "handler", h.name, "commit", position.Commit, "prepare", position.Prepare, "retryCount", retryCount)
			return nil
		}

		h.logger.Error("event handler checkpoint update failed", "handler", h.name, "commit", position.Commit, "prepare", position.Prepare, "retryCount", retryCount, "err", err)
		if waitErr := sleepWithContext(ctx, retryDelay(500*time.Millisecond, 10*time.Second, 500*time.Millisecond, retryCount)); waitErr != nil {
			return waitErr
		}
		retryCount++
	}
}

func retryDelay(base, max, jitter time.Duration, retryCount int) time.Duration {
	multiplier := math.Pow(2, float64(retryCount))
	delay := time.Duration(float64(base) * multiplier)
	if jitter > 0 {
		delay += time.Duration(rand.Int63n(int64(jitter)))
	}
	if delay > max {
		return max
	}
	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
