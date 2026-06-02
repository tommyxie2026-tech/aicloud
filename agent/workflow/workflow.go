package workflow

import (
	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// PolicyChecker is the deterministic policy interface used by workflow.
type PolicyChecker interface {
	Evaluate(p *proposal.ChangeProposal) (proposal.PolicyResult, error)
}

// Planner converts validated model output into evaluated workflow proposals.
type Planner struct {
	checker PolicyChecker
}

func NewPlanner(checker PolicyChecker) *Planner {
	return &Planner{checker: checker}
}

func (p *Planner) BuildEvaluatedProposal(plan schema.ChangePlan, createdBy string) (*proposal.ChangeProposal, error) {
	changeProposal, err := proposal.FromChangePlan(plan, createdBy)
	if err != nil {
		return nil, err
	}
	if p.checker == nil {
		return nil, NewWorkflowError("MissingPolicyChecker", "policy checker is required")
	}

	policyResult, err := p.checker.Evaluate(changeProposal)
	if err != nil {
		return nil, err
	}
	changeProposal.ApplyPolicyResult(policyResult)
	return changeProposal, nil
}

type WorkflowError struct {
	Code    string
	Message string
}

func NewWorkflowError(code string, message string) *WorkflowError {
	return &WorkflowError{Code: code, Message: message}
}

func (e *WorkflowError) Error() string {
	return e.Code + ": " + e.Message
}
