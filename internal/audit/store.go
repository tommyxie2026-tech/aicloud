package audit

import (
	"context"
	"sort"
	"sync"
	"time"
)

type Event struct {
	ID              string            `json:"id"`
	TaskID          string            `json:"taskId"`
	TraceID         string            `json:"traceId"`
	AgentID         string            `json:"agentId"`
	ToolID          string            `json:"toolId"`
	Action          string            `json:"action"`
	PolicyAllowed   bool              `json:"policyAllowed"`
	PolicyVersion   string            `json:"policyVersion,omitempty"`
	MatchedRule     string            `json:"matchedRule,omitempty"`
	ApprovalID      string            `json:"approvalId,omitempty"`
	CredentialLease string            `json:"credentialLease,omitempty"`
	InputDigest     string            `json:"inputDigest,omitempty"`
	ResultDigest    string            `json:"resultDigest,omitempty"`
	Status          string            `json:"status"`
	Reason          string            `json:"reason,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
}

type Store interface {
	Append(context.Context, Event) error
	ListByTask(context.Context, string) ([]Event, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

func (s *MemoryStore) Append(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryStore) ListByTask(_ context.Context, taskID string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Event, 0)
	for _, event := range s.events {
		if event.TaskID == taskID {
			items = append(items, event)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
