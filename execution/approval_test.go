package execution

import (
	"errors"
	"testing"
	"time"
)

func TestApprovalIsBoundToExactExecutionPlanNodeAndResource(t *testing.T) {
	now := time.Now()
	approval := ApprovalRecord{
		ID:                "a1",
		PolicyDecisionRef: "pd1",
		Scope: ApprovalScope{
			ExecutionRef:   "exec-1",
			PlanRevision:   2,
			NodeRef:        "restart-cn",
			Action:         "restart",
			Resource:       "cn/pro03",
			RequestedScope: "production",
		},
		ApprovedBy: "operator-1",
		ApprovedAt: now,
	}

	if err := approval.Authorizes(approval.Scope, now); err != nil {
		t.Fatalf("expected exact scope to be authorized: %v", err)
	}

	changed := approval.Scope
	changed.PlanRevision = 3
	if err := approval.Authorizes(changed, now); !errors.Is(err, ErrApprovalMismatch) {
		t.Fatalf("expected plan revision mismatch, got %v", err)
	}

	changed = approval.Scope
	changed.Resource = "cn/pro04"
	if err := approval.Authorizes(changed, now); !errors.Is(err, ErrApprovalMismatch) {
		t.Fatalf("expected resource mismatch, got %v", err)
	}
}

func TestApprovalExpiresAndCanBeRevoked(t *testing.T) {
	now := time.Now()
	expires := now.Add(time.Minute)
	store := NewMemoryApprovalStore()

	record := ApprovalRecord{
		ID: "a1",
		Scope: ApprovalScope{
			ExecutionRef: "exec-1",
			PlanRevision: 1,
			NodeRef:      "n1",
		},
		ApprovedBy: "operator-1",
		ApprovedAt: now,
		ExpiresAt:  &expires,
	}
	if err := store.Put(record); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("a1")
	if err != nil {
		t.Fatal(err)
	}
	if err := got.Authorizes(record.Scope, now.Add(30*time.Second)); err != nil {
		t.Fatalf("approval should still be active: %v", err)
	}
	if err := got.Authorizes(record.Scope, now.Add(2*time.Minute)); !errors.Is(err, ErrApprovalExpired) {
		t.Fatalf("expected expiry, got %v", err)
	}

	if err := store.Revoke("a1", now.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get("a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveAt(now.Add(20 * time.Second)) {
		t.Fatal("revoked approval must not remain active")
	}
}
