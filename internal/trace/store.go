package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

type Status string

const (
	StatusStarted Status = "STARTED"
	StatusOK      Status = "OK"
	StatusError   Status = "ERROR"
	StatusSkipped Status = "SKIPPED"
)

type Event struct {
	ID           string            `json:"id"`
	TraceID      string            `json:"traceId"`
	TaskID       string            `json:"taskId"`
	SpanID       string            `json:"spanId"`
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	Status       Status            `json:"status"`
	Message      string            `json:"message,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	InputDigest  string            `json:"inputDigest,omitempty"`
	OutputDigest string            `json:"outputDigest,omitempty"`
	StartedAt    time.Time         `json:"startedAt"`
	EndedAt      *time.Time        `json:"endedAt,omitempty"`
}

type Store interface {
	Append(context.Context, Event) error
	ListByTask(context.Context, string) ([]Event, error)
	ListByTrace(context.Context, string) ([]Event, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

func (s *MemoryStore) Append(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, cloneEvent(event))
	return nil
}

func (s *MemoryStore) ListByTask(_ context.Context, taskID string) ([]Event, error) {
	return s.list(func(event Event) bool { return event.TaskID == taskID }), nil
}

func (s *MemoryStore) ListByTrace(_ context.Context, traceID string) ([]Event, error) {
	return s.list(func(event Event) bool { return event.TraceID == traceID }), nil
}

func (s *MemoryStore) list(match func(Event) bool) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Event, 0)
	for _, event := range s.events {
		if match(event) {
			items = append(items, cloneEvent(event))
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartedAt.Equal(items[j].StartedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].StartedAt.Before(items[j].StartedAt)
	})
	return items
}

func NewID(prefix string) string {
	var body [12]byte
	if _, err := rand.Read(body[:]); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(body[:])
}

func cloneEvent(event Event) Event {
	if event.Attributes != nil {
		event.Attributes = cloneMap(event.Attributes)
	}
	return event
}

func cloneMap(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
