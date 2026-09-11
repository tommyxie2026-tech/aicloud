package execution

import "testing"

func TestSupervisorPrepareRunnableDoesNotReserveBudget(t *testing.T) {
	plan := ExecutionPlan{ID: "plan-1", Nodes: []ExecutionNode{{ID: "a", Type: NodeModel}, {ID: "b", Type: NodeTool}}}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 4}})
	s := NewSupervisor(staticPolicy{}, staticResolver{}, staticEstimator{cost: 2}, ledger)

	prepared, conditions, err := s.PrepareRunnable(testExecution(), plan, map[NodeID]NodeState{})
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 2 || len(conditions) != 0 {
		t.Fatalf("unexpected preparation result prepared=%+v conditions=%+v", prepared, conditions)
	}
	for _, candidate := range prepared {
		if candidate.Estimate.Cost != 2 || candidate.Estimate.NodeAttempts != 1 {
			t.Fatalf("estimate not preserved: %+v", candidate)
		}
		if candidate.Policy.Decision != PolicyAllow {
			t.Fatalf("prepared node must carry ALLOW decision: %+v", candidate.Policy)
		}
	}

	snap := ledger.Snapshot()
	if snap.Reserved.Cost != 0 || snap.Reserved.NodeAttempts != 0 {
		t.Fatalf("PrepareRunnable must not reserve budget: %+v", snap.Reserved)
	}
}
