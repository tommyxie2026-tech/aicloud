package pipeline

import (
	"context"

	"github.com/tommyxie2026-tech/aicloud/agent/prdraft"
	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/gateway"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// PlanGateway is the minimum gateway capability required by Pipeline.
type PlanGateway interface {
	GeneratePlan(ctx context.Context, req gateway.GeneratePlanRequest) (*schema.ChangePlan, *gateway.AuditRecord, error)
}

// EvaluatedProposalBuilder converts a validated ChangePlan into a policy-evaluated proposal.
type EvaluatedProposalBuilder interface {
	BuildEvaluatedProposal(plan schema.ChangePlan, createdBy string) (*proposal.ChangeProposal, error)
}

// DraftGenerator converts an evaluated proposal into a reviewable PR draft.
type DraftGenerator interface {
	Generate(p *proposal.ChangeProposal) (*prdraft.Draft, error)
}

// DraftPipeline runs the first end-to-end planning flow.
type DraftPipeline struct {
	gateway        PlanGateway
	planner        EvaluatedProposalBuilder
	draftGenerator DraftGenerator
}

func NewDraftPipeline(gateway PlanGateway, planner EvaluatedProposalBuilder, draftGenerator DraftGenerator) *DraftPipeline {
	return &DraftPipeline{gateway: gateway, planner: planner, draftGenerator: draftGenerator}
}

type Request struct {
	RequestID  string
	UserID     string
	UserIntent string
	RiskHint   string
	CreatedBy  string
}

type Result struct {
	Plan     *schema.ChangePlan
	Audit    *gateway.AuditRecord
	Proposal *proposal.ChangeProposal
	Draft    *prdraft.Draft
}

func (p *DraftPipeline) Run(ctx context.Context, req Request) (*Result, error) {
	if p.gateway == nil {
		return nil, NewPipelineError("MissingGateway", "plan gateway is required")
	}
	if p.planner == nil {
		return nil, NewPipelineError("MissingPlanner", "evaluated proposal planner is required")
	}
	if p.draftGenerator == nil {
		return nil, NewPipelineError("MissingDraftGenerator", "draft generator is required")
	}

	plan, audit, err := p.gateway.GeneratePlan(ctx, gateway.GeneratePlanRequest{RequestID: req.RequestID, UserID: req.UserID, UserIntent: req.UserIntent, RiskHint: req.RiskHint})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, NewPipelineError("MissingPlan", "gateway returned nil plan")
	}

	evaluatedProposal, err := p.planner.BuildEvaluatedProposal(*plan, req.CreatedBy)
	if err != nil {
		return nil, err
	}

	draft, err := p.draftGenerator.Generate(evaluatedProposal)
	if err != nil {
		return nil, err
	}

	return &Result{Plan: plan, Audit: audit, Proposal: evaluatedProposal, Draft: draft}, nil
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
