package gitops

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestBuildPatchPlan(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("spec.workers[name=gpu-workers].replicas")

	plan, err := planner.BuildPatchPlan(changeProposal, "examples/infra/managedcluster-dev-gpu.yaml", "")
	if err != nil {
		t.Fatalf("BuildPatchPlan returned error: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan")
	}
	if plan.SourcePath != "examples/infra/managedcluster-dev-gpu.yaml" {
		t.Fatalf("unexpected source path: %s", plan.SourcePath)
	}
	if plan.OutputPath != plan.SourcePath {
		t.Fatalf("expected output path to default to source path")
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("expected one change, got %d", len(plan.Changes))
	}
	if len(plan.Rollback) != 1 {
		t.Fatalf("expected one rollback change, got %d", len(plan.Rollback))
	}
	if plan.Changes[0].From != 3 || plan.Changes[0].To != 6 {
		t.Fatalf("expected forward change 3 -> 6, got %v -> %v", plan.Changes[0].From, plan.Changes[0].To)
	}
	if plan.Rollback[0].From != 6 || plan.Rollback[0].To != 3 {
		t.Fatalf("expected rollback change 6 -> 3, got %v -> %v", plan.Rollback[0].From, plan.Rollback[0].To)
	}
	if plan.PR.BranchName != "aicloud/request-001/scaleout/dev-gpu-cluster" {
		t.Fatalf("unexpected branch name: %s", plan.PR.BranchName)
	}
	if plan.PR.CommitMessage != "aicloud: ScaleOut ManagedCluster/dev-gpu-cluster" {
		t.Fatalf("unexpected commit message: %s", plan.PR.CommitMessage)
	}
	if plan.PR.Title != "ScaleOut ManagedCluster/dev-gpu-cluster" {
		t.Fatalf("unexpected PR title: %s", plan.PR.Title)
	}
	if plan.PR.Draft {
		t.Fatalf("expected non-draft PR metadata when approval is not required")
	}
}

func TestBuildPatchPlanMarksDraftWhenApprovalRequired(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("spec.workers[name=gpu-workers].replicas")
	changeProposal.ApplyPolicyResult(proposal.PolicyResult{RiskLevel: "High", ApprovalRequired: true, PolicyName: "default-risk-policy", MatchedRule: "prod-or-high-risk", Result: "REVIEW_REQUIRED", Reason: "approval required"})

	plan, err := planner.BuildPatchPlan(changeProposal, "examples/infra/managedcluster-dev-gpu.yaml", "")
	if err != nil {
		t.Fatalf("BuildPatchPlan returned error: %v", err)
	}
	if !plan.PR.Draft {
		t.Fatalf("expected draft PR metadata when approval is required")
	}
}

func TestBuildPatchPlanRejectsUnevaluatedProposal(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("spec.workers[name=gpu-workers].replicas")
	changeProposal.PolicyResult = nil

	_, err := planner.BuildPatchPlan(changeProposal, "examples/infra/managedcluster-dev-gpu.yaml", "")
	if err == nil {
		t.Fatalf("expected policy not evaluated error")
	}
}

func TestBuildPatchPlanRejectsBlockedField(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("status.phase")

	_, err := planner.BuildPatchPlan(changeProposal, "examples/infra/managedcluster-dev-gpu.yaml", "")
	if err == nil {
		t.Fatalf("expected blocked field error")
	}
}

func TestBuildPatchPlanRejectsUnknownField(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("spec.network.cidr")

	_, err := planner.BuildPatchPlan(changeProposal, "examples/infra/managedcluster-dev-gpu.yaml", "")
	if err == nil {
		t.Fatalf("expected field not allowed error")
	}
}

func TestBuildPatchPlanRequiresSourcePath(t *testing.T) {
	planner := NewPatchPlanner()
	changeProposal := validEvaluatedProposal("spec.workers[name=gpu-workers].replicas")

	_, err := planner.BuildPatchPlan(changeProposal, "", "")
	if err == nil {
		t.Fatalf("expected missing source path error")
	}
}

func validEvaluatedProposal(field string) *proposal.ChangeProposal {
	p := &proposal.ChangeProposal{
		ID:            "proposal-001",
		RequestID:     "request-001",
		Intent:        "scale dev-gpu-cluster gpu-workers from 3 to 6",
		Target:        schema.ResourceRef{Kind: "ManagedCluster", Namespace: "default", Name: "dev-gpu-cluster"},
		OperationType: "ScaleOut",
		Environment:   "dev",
		Changes: []proposal.ProposalChange{
			{Field: field, From: 3, To: 6, Reason: "scale out"},
		},
		ModelRiskHint:  "Low",
		Rollback:       proposal.RollbackProposal{Summary: "set gpu-workers replicas back to 3"},
		ValidationPlan: proposal.ValidationPlan{Expected: []string{"workerReadyReplicas=6"}},
	}
	p.ApplyPolicyResult(proposal.PolicyResult{RiskLevel: "Medium", ApprovalRequired: false, PolicyName: "default-risk-policy", MatchedRule: "dev-managedcluster-small-scale", Result: "PASS", Reason: "dev ManagedCluster scale-out within small replica delta"})
	return p
}
