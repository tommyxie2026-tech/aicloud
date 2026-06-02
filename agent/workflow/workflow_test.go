package workflow

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
	"github.com/tommyxie2026-tech/aicloud/policy/checker"
)

func TestBuildEvaluatedProposal(t *testing.T) {
	planner := NewPlanner(checker.NewChecker(checker.DefaultPolicy()))

	proposal, err := planner.BuildEvaluatedProposal(validPlan(), "tester")
	if err != nil {
		t.Fatalf("BuildEvaluatedProposal returned error: %v", err)
	}
	if proposal == nil {
		t.Fatalf("expected proposal")
	}
	if !proposal.IsPolicyEvaluated() {
		t.Fatalf("expected policy evaluated proposal")
	}
	if proposal.PolicyResult.RiskLevel != "Medium" {
		t.Fatalf("expected Medium, got %s", proposal.PolicyResult.RiskLevel)
	}
	if proposal.ApprovalRequired {
		t.Fatalf("expected approvalRequired=false for dev small scale")
	}
}

func TestBuildEvaluatedProposalFailsWithoutPolicyChecker(t *testing.T) {
	planner := NewPlanner(nil)

	proposal, err := planner.BuildEvaluatedProposal(validPlan(), "tester")
	if err == nil {
		t.Fatalf("expected missing policy checker error")
	}
	if proposal != nil {
		t.Fatalf("expected nil proposal on error")
	}
}

func validPlan() schema.ChangePlan {
	return schema.ChangePlan{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindChangePlan,
			RequestID:     "plan-001",
			TaskType:      string(provider.TaskGeneratePlan),
			CreatedBy:     "model-gateway",
		},
		Intent:        "scale dev-gpu-cluster gpu-workers from 3 to 6",
		Target:        schema.ResourceRef{APIVersion: "infra.ai/v1alpha1", Kind: "ManagedCluster", Namespace: "default", Name: "dev-gpu-cluster"},
		OperationType: "ScaleOut",
		Environment:   "dev",
		RiskHint:      "Low",
		Changes: []schema.PlannedChange{
			{Field: "spec.workers[name=gpu-workers].replicas", From: 3, To: 6, Reason: "scale out"},
		},
		Rollback:   schema.RollbackSummary{Summary: "set replicas back to 3"},
		Validation: schema.ValidationExpectations{Expected: []string{"workerReadyReplicas=6"}},
	}
}
