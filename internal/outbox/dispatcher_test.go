package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

type fakeStore struct {
	leased       []repository.LeasedOutboxMessage
	deliveredIDs []string
	failedIDs    []string
	failStatus   domain.OutboxStatus
}

func (s *fakeStore) Lease(context.Context, string, int, time.Time, time.Duration) ([]repository.LeasedOutboxMessage, error) {
	return append([]repository.LeasedOutboxMessage(nil), s.leased...), nil
}
func (s *fakeStore) MarkDelivered(_ context.Context, id, _ string, _ time.Time) error {
	s.deliveredIDs = append(s.deliveredIDs, id)
	return nil
}
func (s *fakeStore) FailDelivery(_ context.Context, id, _ string, _, _ time.Time, _ int, _ string) (domain.OutboxStatus, error) {
	s.failedIDs = append(s.failedIDs, id)
	if s.failStatus == "" {
		return domain.OutboxPending, nil
	}
	return s.failStatus, nil
}

type fakeAdapter struct {
	err  error
	keys []string
}

func (a *fakeAdapter) Deliver(_ context.Context, message domain.OutboxMessage) error {
	a.keys = append(a.keys, message.IdempotencyKey)
	return a.err
}

func TestDispatcherMarksSuccessfulDelivery(t *testing.T) {
	store := &fakeStore{leased: []repository.LeasedOutboxMessage{{
		Message: domain.OutboxMessage{
			OutboxID: "outbox-1", Destination: "workflow.start",
			IdempotencyKey: "delivery-1", Attempts: 1,
		},
	}}}
	dispatcher, err := NewDispatcher(store, "worker-a", time.Minute, 3, time.Second)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	adapter := &fakeAdapter{}
	if err := dispatcher.Register("workflow.start", adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	result, err := dispatcher.DispatchOnce(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if result.Delivered != 1 || len(store.deliveredIDs) != 1 || store.deliveredIDs[0] != "outbox-1" {
		t.Fatalf("unexpected delivery result=%#v ids=%v", result, store.deliveredIDs)
	}
	if len(adapter.keys) != 1 || adapter.keys[0] != "delivery-1" {
		t.Fatalf("adapter did not receive stable idempotency key: %v", adapter.keys)
	}
}

func TestDispatcherRetriesDeliveryFailure(t *testing.T) {
	store := &fakeStore{leased: []repository.LeasedOutboxMessage{{
		Message: domain.OutboxMessage{
			OutboxID: "outbox-retry", Destination: "workflow.start",
			IdempotencyKey: "delivery-retry", Attempts: 1,
		},
	}}}
	dispatcher, _ := NewDispatcher(store, "worker-a", time.Minute, 3, time.Second)
	adapter := &fakeAdapter{err: errors.New("temporary failure")}
	_ = dispatcher.Register("workflow.start", adapter)
	result, err := dispatcher.DispatchOnce(context.Background(), 10)
	if err != nil {
		t.Fatalf("delivery failure should be recorded, not returned as dispatcher failure: %v", err)
	}
	if result.Retried != 1 || len(store.failedIDs) != 1 || store.failedIDs[0] != "outbox-retry" {
		t.Fatalf("unexpected retry result=%#v failed=%v", result, store.failedIDs)
	}
}

func TestDispatcherDeadLettersUnknownDestination(t *testing.T) {
	store := &fakeStore{
		leased: []repository.LeasedOutboxMessage{{
			Message: domain.OutboxMessage{
				OutboxID: "outbox-unknown", Destination: "unknown",
				IdempotencyKey: "delivery-unknown", Attempts: 1,
			},
		}},
		failStatus: domain.OutboxDeadLetter,
	}
	dispatcher, _ := NewDispatcher(store, "worker-a", time.Minute, 3, time.Second)
	result, err := dispatcher.DispatchOnce(context.Background(), 10)
	if !errors.Is(err, ErrDestinationNotConfigured) {
		t.Fatalf("error=%v want ErrDestinationNotConfigured", err)
	}
	if result.DeadLetter != 1 || len(store.failedIDs) != 1 {
		t.Fatalf("unexpected dead-letter result=%#v failed=%v", result, store.failedIDs)
	}
}

func TestDispatcherBackoffIsBounded(t *testing.T) {
	store := &fakeStore{}
	dispatcher, _ := NewDispatcher(store, "worker-a", time.Minute, 10, time.Minute)
	if got := dispatcher.backoff(10); got != 5*time.Minute {
		t.Fatalf("backoff=%s want=5m", got)
	}
}
