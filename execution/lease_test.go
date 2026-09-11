package execution

import (
	"testing"
	"time"
)

func TestNodeLeaseActiveAtUsesClosedOpenInterval(t *testing.T) {
	now := time.Now().UTC()
	lease := NodeLease{
		Token:       "token-1",
		Fence:       1,
		ClaimedAt:   now,
		HeartbeatAt: now,
		ExpiresAt:   now.Add(time.Minute),
	}

	if !lease.ActiveAt(now) {
		t.Fatal("lease should be active at claim time")
	}
	if !lease.ActiveAt(now.Add(30 * time.Second)) {
		t.Fatal("lease should be active before expiry")
	}
	if lease.ActiveAt(lease.ExpiresAt) {
		t.Fatal("lease must be inactive at expiry")
	}
}

func TestNodeRuntimeTerminalStates(t *testing.T) {
	terminal := []NodeState{NodeSucceeded, NodeFailed, NodeSkipped, NodeCancelled}
	for _, state := range terminal {
		if !((NodeRuntimeRecord{State: state}).Terminal()) {
			t.Fatalf("state %s should be terminal", state)
		}
	}
	if (NodeRuntimeRecord{State: NodeRunning}).Terminal() {
		t.Fatal("RUNNING must not be terminal")
	}
}
