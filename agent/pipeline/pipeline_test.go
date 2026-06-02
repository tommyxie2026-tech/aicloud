package pipeline

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/agent/prdraft"
	"github.com/tommyxie2026-tech/aicloud/agent/workflow"
	"github.com/tommyxie2026-tech/aicloud/model/gateway"
	"github.com/tommyxie2026-tech/aicloud/model/mock"
	"github.com/tommyxie2026-tech/aicloud/model/safety"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
	"github.com/tommyxie2026-tech/aicloud/policy/checker"
)

func TestDraftPipelineRun(t *testing.T) {
	modelGateway := gateway.NewGateway(mock.NewProvider(), schema.NewBasicValidator(), safety.NewValidator(), nil)
	planner := workflow.NewPlanner(checker.NewChecker(checker.DefaultPolicy()))
	draftGenerator := prdraft.NewGenerator()
	pipeline := NewDraftPipeline(modelGateway, planner, draftGenerator)

	result, err := pipeline.Run(context.Background(), Request{
		RequestID:  "pipeline-001",
		UserID:     "tester",
		UserIntent: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		RiskHint:   "Medium",
		CreatedBy:  "tester",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.Plan == nil {
		t.Fatalf("expected plan")
	}
	if result.Audit == nil {
		t.Fatalf("expected audit")
	}
	if result.Proposal == nil || !result.Proposal.IsPolicyEvaluated() {
		t.Fatalf("expected evaluated proposal")
	}
	if result.Draft == nil {
		t.Fatalf("expected draft")
	}
	if result.Draft.Title != "ScaleOut ManagedCluster/dev-gpu-cluster" {
		t.Fatalf("unexpected draft title: %s", result.Draft.Title)
	}
}

func TestDraftPipelineRequiresGateway(t *testing.T) {
	pipeline := NewDraftPipeline(nil, workflow.NewPlanner(checker.NewChecker(checker.DefaultPolicy())), prdraft.NewGenerator())

	result, err := pipeline.Run(context.Background(), Request{})
	if err == nil {
		t.Fatalf("expected missing gateway error")
	}
	if result != nil {
		t.Fatalf("expected nil result")
	}
}
