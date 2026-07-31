package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Snapshot struct {
	Key                 string     `json:"key"`
	State               State      `json:"state"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	OpenedAt            *time.Time `json:"openedAt,omitempty"`
	LastFailureAt       *time.Time `json:"lastFailureAt,omitempty"`
	LastSuccessAt       *time.Time `json:"lastSuccessAt,omitempty"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type Store interface {
	Get(context.Context, string) (Snapshot, error)
	Put(context.Context, Snapshot) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	states map[string]Snapshot
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{states: make(map[string]Snapshot)} }

func (s *MemoryStore) Get(_ context.Context, key string) (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, ok := s.states[key]
	if !ok {
		return Snapshot{Key: key, State: StateClosed}, nil
	}
	return snapshot, nil
}

func (s *MemoryStore) Put(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	s.states[snapshot.Key] = snapshot
	s.mu.Unlock()
	return nil
}

type Breaker struct {
	store            Store
	failureThreshold int
	cooldown         time.Duration
	now              func() time.Time
}

func New(store Store, failureThreshold int, cooldown time.Duration) *Breaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	return &Breaker{store: store, failureThreshold: failureThreshold, cooldown: cooldown, now: time.Now}
}

func (b *Breaker) Allow(ctx context.Context, key string) (bool, Snapshot, error) {
	if key == "" {
		return false, Snapshot{}, fmt.Errorf("circuit-breaker key is required")
	}
	snapshot, err := b.store.Get(ctx, key)
	if err != nil {
		return false, Snapshot{}, err
	}
	now := b.now().UTC()
	if snapshot.State == StateOpen && snapshot.OpenedAt != nil && now.Sub(*snapshot.OpenedAt) >= b.cooldown {
		snapshot.State = StateHalfOpen
		snapshot.UpdatedAt = now
		if err := b.store.Put(ctx, snapshot); err != nil {
			return false, Snapshot{}, err
		}
	}
	return snapshot.State != StateOpen, snapshot, nil
}

func (b *Breaker) Success(ctx context.Context, key string) error {
	now := b.now().UTC()
	snapshot, err := b.store.Get(ctx, key)
	if err != nil {
		return err
	}
	snapshot.Key = key
	snapshot.State = StateClosed
	snapshot.ConsecutiveFailures = 0
	snapshot.OpenedAt = nil
	snapshot.LastSuccessAt = &now
	snapshot.UpdatedAt = now
	return b.store.Put(ctx, snapshot)
}

func (b *Breaker) Failure(ctx context.Context, key string) (Snapshot, error) {
	now := b.now().UTC()
	snapshot, err := b.store.Get(ctx, key)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Key = key
	snapshot.ConsecutiveFailures++
	snapshot.LastFailureAt = &now
	snapshot.UpdatedAt = now
	if snapshot.State == StateHalfOpen || snapshot.ConsecutiveFailures >= b.failureThreshold {
		snapshot.State = StateOpen
		snapshot.OpenedAt = &now
	}
	if err := b.store.Put(ctx, snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}
