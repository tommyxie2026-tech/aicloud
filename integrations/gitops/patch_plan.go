package gitops

import (
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/agent/proposal"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// ManifestPatchPlan is a deterministic and reviewable plan for GitOps manifest mutation.
// It does not apply changes to a live cluster.
type ManifestPatchPlan struct {
	RequestID    string
	ProposalID   string
	Target       schema.ResourceRef
	SourcePath   string
	OutputPath   string
	Changes      []ManifestFieldChange
	Rollback     []ManifestFieldChange
	Validation   []string
	PolicyResult proposal.PolicyResult
}

type ManifestFieldChange struct {
	Field  string
	From   any
	To     any
	Reason string
}

type PatchPlanner struct {
	allowedFields map[string]bool
}

func NewPatchPlanner() *PatchPlanner {
	return &PatchPlanner{allowedFields: map[string]bool{
		"spec.workers[name=gpu-workers].replicas": true,
	}}
}

func (p *PatchPlanner) BuildPatchPlan(changeProposal *proposal.ChangeProposal, sourcePath string, outputPath string) (*ManifestPatchPlan, error) {
	if changeProposal == nil {
		return nil, NewGitOpsError("NilProposal", "proposal is nil")
	}
	if !changeProposal.IsPolicyEvaluated() {
		return nil, NewGitOpsError("PolicyNotEvaluated", "proposal must be evaluated by policy before manifest patch planning")
	}
	if sourcePath == "" {
		return nil, NewGitOpsError("MissingSourcePath", "source manifest path is required")
	}
	if outputPath == "" {
		outputPath = sourcePath
	}
	if len(changeProposal.Changes) == 0 {
		return nil, NewGitOpsError("MissingChanges", "proposal changes must not be empty")
	}

	plan := &ManifestPatchPlan{
		RequestID:    changeProposal.RequestID,
		ProposalID:   changeProposal.ID,
		Target:       changeProposal.Target,
		SourcePath:   sourcePath,
		OutputPath:   outputPath,
		Validation:   append([]string{}, changeProposal.ValidationPlan.Expected...),
		PolicyResult: *changeProposal.PolicyResult,
	}

	for _, change := range changeProposal.Changes {
		if err := p.validateField(change.Field); err != nil {
			return nil, err
		}
		fieldChange := ManifestFieldChange{Field: change.Field, From: change.From, To: change.To, Reason: change.Reason}
		plan.Changes = append(plan.Changes, fieldChange)
		plan.Rollback = append(plan.Rollback, ManifestFieldChange{Field: change.Field, From: change.To, To: change.From, Reason: "rollback: " + change.Reason})
	}

	return plan, nil
}

func (p *PatchPlanner) validateField(field string) error {
	if field == "" {
		return NewGitOpsError("MissingField", "change field is required")
	}
	if isBlockedField(field) {
		return NewGitOpsError("BlockedField", fmt.Sprintf("field %s is blocked for GitOps manifest generation", field))
	}
	if !p.allowedFields[field] {
		return NewGitOpsError("FieldNotAllowed", fmt.Sprintf("field %s is not in GitOps allowlist", field))
	}
	return nil
}

func isBlockedField(field string) bool {
	blockedPrefixes := []string{
		"status",
		"metadata.finalizers",
		"metadata.ownerReferences",
		"spec.credentials",
		"spec.secretRef",
		"spec.bmcSecretRef",
	}
	for _, prefix := range blockedPrefixes {
		if field == prefix || strings.HasPrefix(field, prefix+".") {
			return true
		}
	}
	return false
}

type GitOpsError struct {
	Code    string
	Message string
}

func NewGitOpsError(code string, message string) *GitOpsError {
	return &GitOpsError{Code: code, Message: message}
}

func (e *GitOpsError) Error() string {
	return e.Code + ": " + e.Message
}
