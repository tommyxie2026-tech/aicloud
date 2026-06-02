package prdraft

import (
	"strings"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestGenerateDraftFromEvaluatedProposal(t *testing.T) {
	gen := NewGenerator()
	p := validEvaluatedProposal()

	draft, err := gen.Generate(p)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if draft == nil {
		t.Fatalf("expected draft")
	}
	if draft.Title != "ScaleOut ManagedCluster/dev-gpu-cluster" {
		t.Fatalf("unexpected title: %s", draft.Title)
	}
	mustContain(t, draft.Body, "## Intent")
	mustContain(t, draft.Body, "## Proposed Changes")
	mustContain(t, draft.Body, "## Risk and Approval")
	mustContain(t, draft.Body, "Policy risk level: `Medium`")
	mustContain(t, draft.Body, "Approval required: `false`")
	mustContain(t, draft.Body, "## Rollback Plan")
	mustContain(t, draft.Body, "set gpu-workers replicas back to 3")
	mustContain(t, draft.Body, "## Validation Checklist")
	mustContain(t, draft.Body, "- [ ] workerReadyReplicas=6")
	mustContain(t, draft.Body, "No direct infrastructure execution")
}

func TestGenerateDraftRequiresPolicyEvaluation(t *testing.T) {
	gen := NewGenerator()
	p := validEvaluatedProposal()
	p.PolicyResult = nil

	draft, err := gen.Generate(p)
	if err == nil {
		t.Fatalf("expected policy not evaluated error")
	}
	if draft != nil {
		t.Fatalf("expected nil draft")
	}
}

func TestGenerateDraftRejectsNilProposal(t *testing.T) {
	gen := NewGenerator()

	draft, err := gen.Generate(nil)
	if err == nil {
		t.Fatalf("expected nil proposal error")
	}
	if draft != nil {
		t.Fatalf("expected nil draft")
	}
}

func validEvaluatedProposal() *proposal.ChangeProposal {
	p := &proposal.ChangeProposal{
		ID:            "proposal-001",
		RequestID:     "request-001",
		Intent:        "scale dev-gpu-cluster gpu-workers from 3 to 6",
		Target:        schema.ResourceRef{Kind: "ManagedCluster", Namespace: "default", Name: "dev-gpu-cluster"},
		OperationType: "ScaleOut",
		Environment:   "dev",
		Changes: []proposal.ProposalChange{
			{Field: "spec.workers[name=gpu-workers].replicas", From: 3, To: 6, Reason: "scale out"},
		},
		ModelRiskHint:  "Low",
		Rollback:       proposal.RollbackProposal{Summary: "set gpu-workers replicas back to 3"},
		ValidationPlan: proposal.ValidationPlan{Expected: []string{"workerReadyReplicas=6"}},
	}
	p.ApplyPolicyResult(proposal.PolicyResult{RiskLevel: "Medium", ApprovalRequired: false, PolicyName: "default-risk-policy", MatchedRule: "dev-managedcluster-small-scale", Result: "PASS", Reason: "dev ManagedCluster scale-out within small replica delta"})
	return p
}

func mustContain(t *testing.T, text string, expected string) {
	t.Helper()
	if !strings.Contains(text, expected) {
		t.Fatalf("expected body to contain %q, got:\n%s", expected, text)
	}
}
