package approval

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrApprovalNotFound = errors.New("approval not found")

type Record struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	ToolID     string    `json:"toolId"`
	Action     string    `json:"action"`
	ApprovedBy string    `json:"approvedBy"`
	ApprovedAt time.Time `json:"approvedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
	Revoked    bool      `json:"revoked"`
}

type Store interface {
	Get(context.Context, string) (Record, error)
	Put(context.Context, Record) error
	Validate(context.Context, string, string, string, string) (Record, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Record
	now     func() time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: make(map[string]Record), now: time.Now}
}

func (s *MemoryStore) Get(_ context.Context, id string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	if !ok {
		return Record{}, ErrApprovalNotFound
	}
	return record, nil
}

func (s *MemoryStore) Put(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ID] = record
	return nil
}

func (s *MemoryStore) Validate(ctx context.Context, approvalID, taskID, toolID, action string) (Record, error) {
	record, err := s.Get(ctx, approvalID)
	if err != nil {
		return Record{}, err
	}
	if record.Revoked || record.TaskID != taskID || record.ToolID != toolID || record.Action != action || !record.ExpiresAt.After(s.now()) {
		return Record{}, ErrApprovalNotFound
	}
	return record, nil
}
