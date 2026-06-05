package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("event position conflict")
	ErrInvalidEvent = errors.New("invalid event")
)

type Position struct {
	Commit  int64 `json:"commitPosition"`
	Prepare int64 `json:"preparePosition"`
}

var (
	NoEventPosition   = Position{Commit: -1, Prepare: -1}
	LastEventPosition = Position{Commit: 1<<63 - 1, Prepare: 1<<63 - 1}
)

func (p Position) After(other Position) bool {
	if p.Commit != other.Commit {
		return p.Commit > other.Commit
	}
	return p.Prepare > other.Prepare
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Criterion struct {
	Tags []Tag `json:"tags"`
}

type Query struct {
	Criteria []Criterion `json:"criteria"`
}

type DomainEvent struct {
	EventID   string         `json:"eventId"`
	EventType string         `json:"eventType"`
	Data      map[string]any `json:"data"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ResolvedEvent struct {
	Position Position    `json:"position"`
	Event    DomainEvent `json:"event"`
}

type WriteResult struct {
	Position Position `json:"position"`
}

type Saver interface {
	SaveEvents(ctx context.Context, events []DomainEvent, expected Position, scopeEvents []ResolvedEvent, subset Query) (WriteResult, error)
}

type Retriever interface {
	GetEvents(ctx context.Context, from Position, count int, direction Direction, query Query) ([]ResolvedEvent, error)
}

type Subscriber interface {
	SubscribeToEvents(ctx context.Context, subscriberName string, after Position, query Query, handle func(context.Context, ResolvedEvent) error) error
}

type Store interface {
	Saver
	Retriever
	Subscriber
}

type Direction string

const (
	Forward  Direction = "forward"
	Backward Direction = "backward"
)

type Checkpointer interface {
	GetCheckpoint(ctx context.Context, name string) (Position, bool, error)
	UpdateCheckpoint(ctx context.Context, name string, position Position) error
}

type Publisher interface {
	Publish(ctx context.Context, subject string, data any) error
}

type MessageSubscription interface {
	Close() error
}

type MessageSubscriber interface {
	Subscribe(ctx context.Context, subject string, handle func(context.Context, []byte)) (MessageSubscription, error)
}

func Scope(data map[string]any) map[string]any {
	scope, ok := data["scope"].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return scope
}

func MustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func MustData(v any) map[string]any {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return out
}

func MergeScope(events []ResolvedEvent, target DomainEvent) (DomainEvent, error) {
	if target.Data == nil {
		target.Data = map[string]any{}
	}
	targetScope := Scope(target.Data)
	sortedEvents := append([]ResolvedEvent(nil), events...)
	sort.SliceStable(sortedEvents, func(i, j int) bool {
		left := sortedEvents[i].Position
		right := sortedEvents[j].Position
		if left.Commit != right.Commit {
			return left.Commit < right.Commit
		}
		return left.Prepare < right.Prepare
	})
	for _, resolved := range sortedEvents {
		for key, value := range Scope(resolved.Event.Data) {
			if existing, ok := targetScope[key]; ok && !reflect.DeepEqual(existing, value) {
				return target, ErrConflict
			}
			targetScope[key] = value
		}
	}
	target.Data["scope"] = targetScope
	return target, nil
}
