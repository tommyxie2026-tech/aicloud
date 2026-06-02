package checker

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestEvaluateDevSmallScalePassesWithoutApproval(t *testing.T) {
	checker := NewChecker(DefaultPolicy())
	p := validProposal("dev", 3, 6, "spec.workers[name=gpu-workers].replicas")

	result, err := checker.Evaluate(p)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Result != "PASS" {
		t.Fatalf("expected PASS, got %s", result.Result)
	}
	if result.RiskLevel != string(RiskMedium) {
		t.Fatalf("expected Medium, got %s", result.RiskLevel)
	}
	if result.ApprovalRequired {
		t.Fatalf("expected approvalRequired=false")
	}
	if result.MatchedRule != "dev-managedcluster-small-scale" {
		t.Fatalf("unexpected rule: %s", result.MatchedRule)
	}
}

func TestEvaluateStagingSmallScaleRequiresApproval(t *testing.T) {
	checker := NewChecker(DefaultPolicy())
	p := validProposal("staging", 3, 6, "spec.workers[name=gpu-workers].replicas")

	result, err := checker.Evaluate(p)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Result != "PASS" {
		t.Fatalf("expected PASS, got %s", result.Result)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approvalRequired=true")
	}
}

func TestEvaluateUnknownFieldFailsClosed(t *testing.T) {
	checker := NewChecker(DefaultPolicy())
	p := validProposal("dev", 3, 6, "spec.network.cidr")

	result, err := checker.Evaluate(p)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Result != "REVIEW_REQUIRED" {
		t.Fatalf("expected REVIEW_REQUIRED, got %s", result.Result)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval required for fail closed")
	}
	if result.MatchedRule != "fail-closed" {
		t.Fatalf("expected fail-closed, got %s", result.MatchedRule)
	}
}

func TestEvaluateLargeDeltaFailsClosed(t *testing.T) {
	checker := NewChecker(DefaultPolicy())
	p := validProposal("dev", 3, 10, "spec.workers[name=gpu-workers].replicas")

	result, err := checker.Evaluate(p)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.MatchedRule != "fail-closed" {
		t.Fatalf("expected fail-closed, got %s", result.MatchedRule)
	}
	if !result.ApprovalRequired {
		t.Fatalf("expected approval required")
	}
}

func TestEvaluateNilProposalReturnsError(t *testing.T) {
	checker := NewChecker(DefaultPolicy())

	_, err := checker.Evaluate(nil)
	if err == nil {
		t.Fatalf("expected nil proposal error")
	}
}

func validProposal(environment string, from int, to int, field string) *proposal.ChangeProposal {
	return &proposal.ChangeProposal{
		ID:            "proposal-001",
		RequestID:     "request-001",
		Intent:        "scale dev-gpu-cluster gpu-workers",
		Target:        schema.ResourceRef{Kind: "ManagedCluster", Name: "dev-gpu-cluster"},
		OperationType: "ScaleOut",
		Environment:   environment,
		Changes: []proposal.ProposalChange{
			{Field: field, From: from, To: to, Reason: "scale out"},
		},
		ModelRiskHint: "Low",
	}
}
