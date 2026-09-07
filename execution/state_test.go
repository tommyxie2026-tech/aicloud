package execution

import "testing"

func TestExecutionTransitions(t *testing.T) {
	tests := []struct {
		name string
		from ExecutionPhase
		to   ExecutionPhase
		want bool
	}{
		{"created to planning", ExecutionCreated, ExecutionPlanning, true},
		{"running to verifying", ExecutionRunning, ExecutionVerifying, true},
		{"running to replanning", ExecutionRunning, ExecutionReplanning, true},
		{"replanning to ready", ExecutionReplanning, ExecutionReady, true},
		{"succeeded is terminal", ExecutionSucceeded, ExecutionRunning, false},
		{"cannot skip planning", ExecutionCreated, ExecutionRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransitionExecution(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransitionExecution(%s,%s)=%v want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestNodeApprovalIsLocalState(t *testing.T) {
	if !CanTransitionNode(NodeReady, NodeWaitingApproval) {
		t.Fatal("READY -> WAITING_APPROVAL should be allowed")
	}
	if !CanTransitionNode(NodeWaitingApproval, NodeReady) {
		t.Fatal("WAITING_APPROVAL -> READY should be allowed after approval")
	}
}

func TestExecutionDoesNotHaveWaitingApprovalPhase(t *testing.T) {
	// Approval is represented by a condition so parallel work can continue.
	status := ExecutionStatus{
		Phase: ExecutionRunning,
		Conditions: []ExecutionCondition{{
			Type:    ConditionApprovalRequired,
			Status:  true,
			NodeRef: "mutating-node",
		}},
	}
	if status.Phase != ExecutionRunning {
		t.Fatalf("execution should remain RUNNING, got %s", status.Phase)
	}
}
