package execution

import (
    "errors"
    "fmt"
    "sync"
    "time"
)

var (
    ErrApprovalNotFound = errors.New("approval not found")
    ErrApprovalExpired  = errors.New("approval expired")
    ErrApprovalMismatch = errors.New("approval scope mismatch")
)

type ApprovalID string

type ApprovalScope struct {
    ExecutionRef  ExecutionID
    PlanRevision  int64
    NodeRef       NodeID
    Action        string
    Resource      string
    RequestedScope string
}

type ApprovalRecord struct {
    ID                ApprovalID
    PolicyDecisionRef PolicyDecisionID
    Scope             ApprovalScope
    ApprovedBy        string
    ApprovedAt        time.Time
    ExpiresAt         *time.Time
    RevokedAt         *time.Time
}

func (a ApprovalRecord) ActiveAt(now time.Time) bool {
    if a.RevokedAt != nil {
        return false
    }
    if a.ExpiresAt != nil && !now.Before(*a.ExpiresAt) {
        return false
    }
    return true
}

func (a ApprovalRecord) Authorizes(scope ApprovalScope, now time.Time) error {
    if !a.ActiveAt(now) {
        return ErrApprovalExpired
    }
    if a.Scope != scope {
        return fmt.Errorf("%w: approved=%+v requested=%+v", ErrApprovalMismatch, a.Scope, scope)
    }
    return nil
}

type ApprovalStore interface {
    Put(record ApprovalRecord) error
    Get(id ApprovalID) (ApprovalRecord, error)
    Revoke(id ApprovalID, at time.Time) error
}

type MemoryApprovalStore struct {
    mu      sync.RWMutex
    records map[ApprovalID]ApprovalRecord
}

func NewMemoryApprovalStore() *MemoryApprovalStore {
    return &MemoryApprovalStore{records: map[ApprovalID]ApprovalRecord{}}
}

func (s *MemoryApprovalStore) Put(record ApprovalRecord) error {
    if record.ID == "" {
        return errors.New("approval id is required")
    }
    if record.Scope.ExecutionRef == "" || record.Scope.NodeRef == "" || record.Scope.PlanRevision <= 0 {
        return errors.New("approval execution, plan revision and node scope are required")
    }
    if record.ApprovedBy == "" || record.ApprovedAt.IsZero() {
        return errors.New("approval actor and timestamp are required")
    }

    s.mu.Lock()
    defer s.mu.Unlock()
    if _, exists := s.records[record.ID]; exists {
        return fmt.Errorf("approval %q already exists", record.ID)
    }
    s.records[record.ID] = record
    return nil
}

func (s *MemoryApprovalStore) Get(id ApprovalID) (ApprovalRecord, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    record, ok := s.records[id]
    if !ok {
        return ApprovalRecord{}, ErrApprovalNotFound
    }
    return record, nil
}

func (s *MemoryApprovalStore) Revoke(id ApprovalID, at time.Time) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    record, ok := s.records[id]
    if !ok {
        return ErrApprovalNotFound
    }
    if record.RevokedAt != nil {
        return nil
    }
    record.RevokedAt = &at
    s.records[id] = record
    return nil
}
