package pipeline

import (
	"context"

	"github.com/tommyxie2026-tech/aicloud/agent/prdraft"
	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/agent/workflow"
	"github.com/tommyxie2026-tech/aicloud/model/gateway"
)

// ModelGateway is the minimum gateway capability required by Pipeline.
type ModelGateway interface {
	GeneratePlan(ctx context.Context, req gateway.GeneratePlanRequest) (*schemaChangePlan, *gateway.AuditRecord, error)
}

// schemaChangePlan aliases the concrete schema type through assignment in constructor wrappers.
// This keeps the pipeline package focused on orchestration boundaries.
type schemaChangePlan = interface{}

// PlanGateway is the concrete gateway adapter used by the default pipeline.
type PlanGateway interface {
	GeneratePlan(ctx context.Context, req gateway.GeneratePlanRequest) (*gatewayCompatibleChangePlan, *gateway.AuditRecord, error)
}

type gatewayCompatibleChangePlan = interface{}

// GatewayGeneratePlanFunc is a testable adapter for gateway.GeneratePlan.
type GatewayGeneratePlanFunc func(ctx context.Context, req gateway.GeneratePlanRequest) (*proposalInput, *gateway.AuditRecord, error)

type proposalInput = interface{}

// DraftPipeline runs the first end-to-end planning flow.
type DraftPipeline struct {
	planGenerator PlanGenerator
	planner       EvaluatedProposalBuilder
	draftGenerator DraftGenerator
}

type PlanGenerator interface {
	GeneratePlan(ctx context.Context, req gateway.GeneratePlanRequest) (*PlanOutput, error)
}

type EvaluatedProposalBuilder interface {
	BuildEvaluatedProposalFromPlan(plan *PlanOutput, createdBy string) (*proposal.ChangeProposal, error)
}

type DraftGenerator interface {
	Generate(p *proposal.ChangeProposal) (*prdraft.Draft, error)
}

type PlanOutput struct {
	Plan  any
	Audit *gateway.AuditRecord
}

type Request struct {
	RequestID  string
	UserID     string
	UserIntent string
	RiskHint   string
	CreatedBy  string
}

type Result struct {
	Plan          any
	Audit         *gateway.AuditRecord
	Proposal      *proposal.ChangeProposal
	Draft         *prdraft.Draft
}

func NewDraftPipeline(planGenerator PlanGenerator, planner EvaluatedProposalBuilder, draftGenerator DraftGenerator) *DraftPipeline {
	return &DraftPipeline{planGenerator: planGenerator, planner: planner, draftGenerator: draftGenerator}
}

func (p *DraftPipeline) Run(ctx context.Context, req Request) (*Result, error) {
	if p.planGenerator == nil {
		return nil, NewPipelineError("MissingPlanGenerator", "plan generator is required")
	}
	if p.planner == nil {
		return nil, NewPipelineError("MissingPlanner", "evaluated proposal planner is required")
	}
	if p.draftGenerator == nil {
		return nil, NewPipelineError("MissingDraftGenerator", "draft generator is required")
	}

	planOutput, err := p.planGenerator.GeneratePlan(ctx, gateway.GeneratePlanRequest{RequestID: req.RequestID, UserID: req.UserID, UserIntent: req.UserIntent, RiskHint: req.RiskHint})
	if err != nil {
		return nil, err
	}

	evaluatedProposal, err := p.planner.BuildEvaluatedProposalFromPlan(planOutput, req.CreatedBy)
	if err != nil {
		return nil, err
	}

	draft, err := p.draftGenerator.Generate(evaluatedProposal)
	if err != nil {
		return nil, err
	}

	return &Result{Plan: planOutput.Plan, Audit: planOutput.Audit, Proposal: evaluatedProposal, Draft: draft}, nil
}

type PipelineError struct {
	Code    string
	Message string
}

func NewPipelineError(code string, message string) *PipelineError {
	return &PipelineError{Code: code, Message: message}
}

func (e *PipelineError) Error() string {
	return e.Code + ": " + e.Message
}

// TODO: replace this file with a typed pipeline after schema package boundaries are finalized.
var _ = workflow.NewPlanner
