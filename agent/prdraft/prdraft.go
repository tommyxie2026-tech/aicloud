package prdraft

import (
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
)

// Draft is a reviewable pull request draft generated from an evaluated proposal.
type Draft struct {
	Title string
	Body  string
}

// Generator converts evaluated ChangeProposal into PR title and body.
type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(p *proposal.ChangeProposal) (*Draft, error) {
	if p == nil {
		return nil, NewDraftError("NilProposal", "proposal is nil")
	}
	if !p.IsPolicyEvaluated() {
		return nil, NewDraftError("PolicyNotEvaluated", "proposal must be evaluated by policy before PR draft generation")
	}
	if len(p.Changes) == 0 {
		return nil, NewDraftError("MissingChanges", "proposal changes must not be empty")
	}

	title := fmt.Sprintf("%s %s/%s", p.OperationType, p.Target.Kind, p.Target.Name)
	body := strings.Join([]string{
		"## Intent",
		p.Intent,
		"",
		"## Target",
		fmt.Sprintf("- Kind: `%s`", p.Target.Kind),
		fmt.Sprintf("- Name: `%s`", p.Target.Name),
		fmt.Sprintf("- Namespace: `%s`", p.Target.Namespace),
		fmt.Sprintf("- Environment: `%s`", p.Environment),
		"",
		"## Proposed Changes",
		renderChanges(p.Changes),
		"",
		"## Risk and Approval",
		fmt.Sprintf("- Model risk hint: `%s`", p.ModelRiskHint),
		fmt.Sprintf("- Policy risk level: `%s`", p.PolicyResult.RiskLevel),
		fmt.Sprintf("- Approval required: `%v`", p.PolicyResult.ApprovalRequired),
		fmt.Sprintf("- Policy: `%s`", p.PolicyResult.PolicyName),
		fmt.Sprintf("- Matched rule: `%s`", p.PolicyResult.MatchedRule),
		fmt.Sprintf("- Policy result: `%s`", p.PolicyResult.Result),
		fmt.Sprintf("- Reason: %s", p.PolicyResult.Reason),
		"",
		"## Rollback Plan",
		renderRollback(p.Rollback),
		"",
		"## Validation Checklist",
		renderValidation(p.ValidationPlan),
		"",
		"## Safety Boundary",
		"- This PR draft is generated from a policy-evaluated proposal.",
		"- No direct infrastructure execution is performed by the model or agent.",
		"- Apply path should remain GitOps / controller based.",
	}, "\n")

	return &Draft{Title: title, Body: body}, nil
}

func renderChanges(changes []proposal.ProposalChange) string {
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		lines = append(lines, fmt.Sprintf("- `%s`: `%v` -> `%v`", change.Field, change.From, change.To))
		if change.Reason != "" {
			lines = append(lines, fmt.Sprintf("  - Reason: %s", change.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func renderRollback(rollback proposal.RollbackProposal) string {
	if rollback.Summary == "" {
		return "- Rollback plan is not provided."
	}
	return "- " + rollback.Summary
}

func renderValidation(plan proposal.ValidationPlan) string {
	if len(plan.Expected) == 0 {
		return "- [ ] Manual validation required."
	}
	lines := make([]string, 0, len(plan.Expected))
	for _, expected := range plan.Expected {
		lines = append(lines, "- [ ] "+expected)
	}
	return strings.Join(lines, "\n")
}

type DraftError struct {
	Code    string
	Message string
}

func NewDraftError(code string, message string) *DraftError {
	return &DraftError{Code: code, Message: message}
}

func (e *DraftError) Error() string {
	return e.Code + ": " + e.Message
}
