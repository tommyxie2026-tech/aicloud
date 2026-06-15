package openai

import (
	"context"
	"testing"
	"time"
)

func TestTimeoutPolicyUsesProvidedTimeout(t *testing.T) {
	policy := NewTimeoutPolicy(15)
	if policy.TimeoutSeconds != 15 {
		t.Fatalf("expected 15 seconds, got %d", policy.TimeoutSeconds)
	}
	if policy.Duration() != 15*time.Second {
		t.Fatalf("unexpected duration: %s", policy.Duration())
	}
}

func TestTimeoutPolicyDefaultsInvalidTimeout(t *testing.T) {
	policy := NewTimeoutPolicy(0)
	if policy.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Fatalf("expected default timeout, got %d", policy.TimeoutSeconds)
	}
}

func TestTimeoutPolicyWithTimeoutHandlesNilParent(t *testing.T) {
	policy := NewTimeoutPolicy(1)
	ctx, cancel := policy.WithTimeout(nil)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatalf("expected deadline")
	}
}

func TestTimeoutPolicyWithTimeoutCreatesDeadline(t *testing.T) {
	policy := NewTimeoutPolicy(1)
	ctx, cancel := policy.WithTimeout(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("expected future deadline")
	}
}

func TestTimeoutPolicyHasDeadline(t *testing.T) {
	policy := NewTimeoutPolicy(1)
	if policy.HasDeadline(context.Background()) {
		t.Fatalf("did not expect deadline on background context")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !policy.HasDeadline(ctx) {
		t.Fatalf("expected deadline")
	}
	if policy.HasDeadline(nil) {
		t.Fatalf("nil context should not have deadline")
	}
}
