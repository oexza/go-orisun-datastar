package eventstore

import (
	"reflect"
	"testing"
)

func TestWithScopeEventIDsStoresDeterministicMetadataWithoutChangingEventData(t *testing.T) {
	t.Parallel()

	target := DomainEvent{
		EventID:   "target-id",
		EventType: "TargetEvent",
		Data:      map[string]any{"scope": map[string]any{"targetId": "target-id"}},
		Metadata:  map[string]any{"source": "test", "scope_event_ids": []string{"untrusted-id"}},
	}
	later := ResolvedEvent{
		Position: Position{Commit: 20, Prepare: 19},
		Event: DomainEvent{
			EventID: "later-id",
			Data:    map[string]any{"scope": map[string]any{"targetId": "conflicting-id"}},
		},
	}
	earlier := ResolvedEvent{
		Position: Position{Commit: 10, Prepare: 9},
		Event: DomainEvent{
			EventID: "earlier-id",
			Data:    map[string]any{"scope": map[string]any{"inheritedId": "not-copied"}},
		},
	}

	result := withScopeEventIDs(target, []ResolvedEvent{later, earlier, earlier})

	wantData := map[string]any{"scope": map[string]any{"targetId": "target-id"}}
	if !reflect.DeepEqual(result.Data, wantData) {
		t.Fatalf("event data changed: got %#v, want %#v", result.Data, wantData)
	}
	if !reflect.DeepEqual(target.Data, wantData) {
		t.Fatalf("input event data changed: got %#v, want %#v", target.Data, wantData)
	}
	if got := target.Metadata["scope_event_ids"]; !reflect.DeepEqual(got, []string{"untrusted-id"}) {
		t.Fatalf("input metadata changed: got %#v", got)
	}
	if got := result.Metadata["scope_event_ids"]; !reflect.DeepEqual(got, []string{"earlier-id", "later-id"}) {
		t.Fatalf("scope event ids: got %#v", got)
	}
	if got := result.Metadata["source"]; got != "test" {
		t.Fatalf("metadata source: got %#v", got)
	}
}

func TestWithScopeEventIDsStoresEmptyList(t *testing.T) {
	t.Parallel()

	result := withScopeEventIDs(DomainEvent{}, nil)

	ids, ok := result.Metadata["scope_event_ids"].([]string)
	if !ok {
		t.Fatalf("scope event ids have type %T, want []string", result.Metadata["scope_event_ids"])
	}
	if len(ids) != 0 {
		t.Fatalf("scope event ids: got %#v, want empty", ids)
	}
}
