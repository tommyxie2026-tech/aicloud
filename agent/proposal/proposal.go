package proposal

import (
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// ChangeProposal is a workflow-ready representation of a validated ChangePlan.
//
// It is still not executable. It is intended to be consumed by policy checks,
// human review, PR generation, and later GitOps/controller workflows.
type ChangeProposal struct {
	ID               string
	RequestID        string
	Intent           string
	Target           schema.ResourceRef
	OperationType    string
	Environment      string
	Changes          []ProposalChange
	ModelRiskHint    string
	PolicyResult     *PolicyResult
	Rollback         RollbackProposal
	ValidationPlan   ValidationPlan
	ApprovalRequired bool
	CreatedBy        string
	CreatedAt        time.Time
}

type ProposalChange struct {
	Field  string
	From   any
	To     any
	Reason string
}

type PolicyResult struct {
	RiskLevel        string
	ApprovalRequired bool
	PolicyName       string
	MatchedRule      string
	Result           string
	Reason           string
}

type RollbackProposal struct {
	Summary string
}

type ValidationPlan struct {
	Expected []string
}

// FromChangePlan converts a validated model ChangePlan into a workflow proposal.
//
// The model risk hint is copied only as a hint. The authoritative risk and
// approval decision must be set later by deterministic policy.
func FromChangePlan(plan schema.ChangePlan, createdBy string) (*ChangeProposal, error) {
	if plan.RequestID == "" {
		return nil, NewProposalError("MissingRequestID", "ChangePlan.requestId is required")
	}
	if plan.Intent == "" {
		return nil, NewProposalError("MissingIntent", "ChangePlan.intent is required")
	}
	if plan.Target.Kind == "" || plan.Target.Name == "" {
		return nil, NewProposalError("MissingTarget", "ChangePlan.target.kind and target.name are required")
	}
	if len(plan.Changes) == 0 {
		return nil, NewProposalError("MissingChanges", "ChangePlan.changes must not be empty")
	}
	if createdBy == "" {
		createdBy = "system"
	}

	proposal := &ChangeProposal{
		ID:             newProposalID(plan.RequestID),
		RequestID:      plan.RequestID,
		Intent:         plan.Intent,
		Target:         plan.Target,
		OperationType:  plan.OperationType,
		Environment:    plan.Environment,
		ModelRiskHint:  plan.RiskHint,
		Rollback:       RollbackProposal{Summary: plan.Rollback.Summary},
		ValidationPlan: ValidationPlan{Expected: append([]string{}, plan.Validation.Expected...)},
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
	}

	for _, change := range plan.Changes {
		proposal.Changes = append(proposal.Changes, ProposalChange{Field: change.Field, From: change.From, To: change.To, Reason: change.Reason})
	}

	return proposal, nil
}

func (p *ChangeProposal) ApplyPolicyResult(result PolicyResult) {
	p.PolicyResult = &result
	p.ApprovalRequired = result.ApprovalRequired
}

func (p *ChangeProposal) IsPolicyEvaluated() bool {
	return p != nil && p.PolicyResult != nil && p.PolicyResult.Result != ""
}

func newProposalID(requestID string) string {
	return fmt.Sprintf("proposal-%s", requestID)
}

type ProposalError struct {
	Code    string
	Message string
}

func NewProposalError(code string, message string) *ProposalError {
	return &ProposalError{Code: code, Message: message}
}

func (e *ProposalError) Error() string {
	return e.Code + ": " + e.Message
}
