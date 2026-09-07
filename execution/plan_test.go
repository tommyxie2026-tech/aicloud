package execution

import "testing"

func TestValidatePlanAcceptsBoundedDAG(t *testing.T) {
	plan := ExecutionPlan{
		ID:       "plan-1",
		GoalRef:  "goal-1",
		Revision: 1,
		Nodes: []ExecutionNode{
			{ID: "collect", Type: NodeTool, Effect: EffectSpec{Class: EffectReadOnly}},
			{ID: "analyze", Type: NodeModel, DependsOn: []NodeID{"collect"}, Effect: EffectSpec{Class: EffectPure}},
			{ID: "verify", Type: NodeVerifier, DependsOn: []NodeID{"analyze"}, Effect: EffectSpec{Class: EffectPure}},
		},
	}
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("ValidatePlan() unexpected error: %v", err)
	}
}

func TestValidatePlanRejectsCycle(t *testing.T) {
	plan := ExecutionPlan{
		ID:       "plan-1",
		GoalRef:  "goal-1",
		Revision: 1,
		Nodes: []ExecutionNode{
			{ID: "a", Type: NodeTool, DependsOn: []NodeID{"b"}, Effect: EffectSpec{Class: EffectReadOnly}},
			{ID: "b", Type: NodeTool, DependsOn: []NodeID{"a"}, Effect: EffectSpec{Class: EffectReadOnly}},
		},
	}
	if err := ValidatePlan(plan); err == nil {
		t.Fatal("ValidatePlan() expected cycle error")
	}
}

func TestValidateEffectSpecRequiresIdempotencyKey(t *testing.T) {
	effect := EffectSpec{Class: EffectIdempotentMutation, RetrySafe: true}
	if err := ValidateEffectSpec(effect); err == nil {
		t.Fatal("expected idempotency key validation error")
	}
}

func TestValidateEffectSpecRejectsRetrySafeNonIdempotentMutation(t *testing.T) {
	effect := EffectSpec{Class: EffectNonIdempotentMutation, RetrySafe: true}
	if err := ValidateEffectSpec(effect); err == nil {
		t.Fatal("expected non-idempotent retry validation error")
	}
}
