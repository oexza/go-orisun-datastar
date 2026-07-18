package auth

import (
	"context"
	"testing"
	"time"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

func TestLoadValidateEmailVerificationOTPContextKeepsEveryFetchedEvent(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	retriever := &stabilizingOTPRetriever{
		generated: eventstore.ResolvedEvent{
			Position: eventstore.Position{Commit: 2, Prepare: 2},
			Event:    NewEmailVerificationOTPGeneratedEvent("otp-1", "123456", expiresAt, "user-1", nil),
		},
		registered: eventstore.ResolvedEvent{
			Position: eventstore.Position{Commit: 1, Prepare: 1},
			Event: eventstore.DomainEvent{
				EventID:   "user-1",
				EventType: UserRegistered,
				Data:      map[string]any{UserRegisteredIDField: "user-1"},
			},
		},
	}

	model, err := loadValidateEmailVerificationOTPContext(context.Background(), ValidateEmailVerificationOTPCommand{
		User: views.User{UserRegisteredID: "user-1"},
		Code: "123456",
	}, retriever)
	if err != nil {
		t.Fatalf("load context: %v", err)
	}

	if retriever.calls != 2 {
		t.Fatalf("retrieval calls: got %d, want 2", retriever.calls)
	}
	if len(model.events) != 4 {
		t.Fatalf("fetched events passed to saver: got %d, want 4", len(model.events))
	}
	wantIDs := []string{"user-1", "otp-1", "user-1", "otp-1"}
	for index, want := range wantIDs {
		if got := model.events[index].Event.EventID; got != want {
			t.Fatalf("fetched event %d: got %q, want %q", index, got, want)
		}
	}
}

type stabilizingOTPRetriever struct {
	calls      int
	generated  eventstore.ResolvedEvent
	registered eventstore.ResolvedEvent
}

func (r *stabilizingOTPRetriever) GetEvents(context.Context, eventstore.Position, int, eventstore.Direction, eventstore.Query) ([]eventstore.ResolvedEvent, error) {
	return nil, nil
}

func (r *stabilizingOTPRetriever) GetLatestByCriteria(context.Context, []eventstore.Criterion) (eventstore.LatestByCriteriaResult, error) {
	r.calls++
	return eventstore.LatestByCriteriaResult{
		Results: []eventstore.LatestCriterionResult{
			{Event: &r.generated},
			{Event: &r.registered},
		},
		ContextPosition: r.generated.Position,
	}, nil
}
