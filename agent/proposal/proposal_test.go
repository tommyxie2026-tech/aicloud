package proposal

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestFromChangePlanCreatesProposal(t *testing.T) {
	plan := validPlan()

	proposal, err := FromChangePlan(plan, "tester")
	if err != nil {
		t.Fatalf("FromChangePlan returned error: %v", err)
	}
	if proposal == nil {
		t.Fatalf("expected proposal")
	}
	if proposal.ID != "proposal-plan-001" {
		t.Fatalf("unexpected proposal id: %s", proposal.ID)
	}
	if proposal.Target.Name != "dev-gpu-cluster" {
		t.Fatalf("expected dev-gpu-cluster, got %s", proposal.Target.Name)
	}
	if len(proposal.Changes) != 1 {
		t.Fatalf("expected one change, got %d", len(proposal.Changes))
	}
	if proposal.ModelRiskHint != "Medium" {
		t.Fatalf("expected model risk hint Medium, got %s", proposal.ModelRiskHint)
	}
	if proposal.PolicyResult != nil {
		t.Fatalf("policy result must not be copied from model output")
	}
	if proposal.ApprovalRequired {
		t.Fatalf("approvalRequired must not be set before policy evaluation")
	}
}

func TestApplyPolicyResult(t *testing.T) {
	proposal, err := FromChangePlan(validPlan(), "tester")
	if err != nil {
		t.Fatalf("FromChangePlan returned error: %v", err)
	}

	proposal.ApplyPolicyResult(PolicyResult{RiskLevel: "Medium", ApprovalRequired: true, PolicyName: "default", MatchedRule: "dev-scale", Result: "PASS", Reason: "requires approval for test"})

	if !proposal.IsPolicyEvaluated() {
		t.Fatalf("expected policy evaluated")
	}
	if !proposal.ApprovalRequired {
		t.Fatalf("expected approval required")
	}
	if proposal.PolicyResult.RiskLevel != "Medium" {
		t.Fatalf("expected Medium, got %s", proposal.PolicyResult.RiskLevel)
	}
}

func TestFromChangePlanRejectsMissingChanges(t *testing.T) {
	plan := validPlan()
	plan.Changes = nil

	_, err := FromChangePlan(plan, "tester")
	if err == nil {
		t.Fatalf("expected missing changes error")
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
		RiskHint:      "Medium",
		Changes: []schema.PlannedChange{
			{Field: "spec.workers[name=gpu-workers].replicas", From: 3, To: 6, Reason: "scale out"},
		},
		Rollback:   schema.RollbackSummary{Summary: "set replicas back to 3"},
		Validation: schema.ValidationExpectations{Expected: []string{"workerReadyReplicas=6"}},
	}
}
