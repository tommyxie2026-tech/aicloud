package circuitbreaker

import (
	"context"
	"testing"
	"time"
)

func TestBreakerOpensAndRecoversThroughHalfOpen(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	breaker := New(NewMemoryStore(), 2, time.Minute)
	breaker.now = func() time.Time { return now }

	if _, err := breaker.Failure(context.Background(), "model-1"); err != nil {
		t.Fatalf("first failure: %v", err)
	}
	snapshot, err := breaker.Failure(context.Background(), "model-1")
	if err != nil {
		t.Fatalf("second failure: %v", err)
	}
	if snapshot.State != StateOpen {
		t.Fatalf("state = %s", snapshot.State)
	}
	allowed, _, err := breaker.Allow(context.Background(), "model-1")
	if err != nil {
		t.Fatalf("Allow while open: %v", err)
	}
	if allowed {
		t.Fatal("open circuit should deny execution")
	}

	now = now.Add(2 * time.Minute)
	allowed, snapshot, err = breaker.Allow(context.Background(), "model-1")
	if err != nil {
		t.Fatalf("Allow after cooldown: %v", err)
	}
	if !allowed || snapshot.State != StateHalfOpen {
		t.Fatalf("expected half-open probe, allowed=%t state=%s", allowed, snapshot.State)
	}
	if err := breaker.Success(context.Background(), "model-1"); err != nil {
		t.Fatalf("Success: %v", err)
	}
	_, snapshot, err = breaker.Allow(context.Background(), "model-1")
	if err != nil {
		t.Fatalf("Allow after success: %v", err)
	}
	if snapshot.State != StateClosed || snapshot.ConsecutiveFailures != 0 {
		t.Fatalf("unexpected recovered state: %#v", snapshot)
	}
}
