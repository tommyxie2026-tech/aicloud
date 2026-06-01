package schema

import "fmt"

// BasicValidator validates required fields and schema-level invariants.
// Safety-specific checks are handled by the safety package.
type BasicValidator struct{}

func NewBasicValidator() *BasicValidator {
	return &BasicValidator{}
}

func (v *BasicValidator) ValidateChangePlan(plan *ChangePlan) error {
	if plan == nil {
		return NewValidationError("ChangePlanNil", "change plan is nil")
	}
	if err := validateCommon(plan.CommonMetadata, KindChangePlan); err != nil {
		return err
	}
	if plan.Intent == "" {
		return NewValidationError("MissingIntent", "ChangePlan.intent is required")
	}
	if err := validateResourceRef(plan.Target, "ChangePlan.target"); err != nil {
		return err
	}
	if plan.OperationType == "" {
		return NewValidationError("MissingOperationType", "ChangePlan.operationType is required")
	}
	if len(plan.Changes) == 0 {
		return NewValidationError("MissingChanges", "ChangePlan.changes must not be empty")
	}
	for i, change := range plan.Changes {
		if change.Field == "" {
			return NewValidationError("MissingChangeField", fmt.Sprintf("ChangePlan.changes[%d].field is required", i))
		}
		if change.To == nil {
			return NewValidationError("MissingChangeTo", fmt.Sprintf("ChangePlan.changes[%d].to is required", i))
		}
	}
	if plan.Rollback.Summary == "" {
		return NewValidationError("MissingRollback", "ChangePlan.rollback.summary is required")
	}
	return nil
}

func (v *BasicValidator) ValidateYamlPatchProposal(patch *YamlPatchProposal) error {
	if patch == nil {
		return NewValidationError("YamlPatchProposalNil", "yaml patch proposal is nil")
	}
	if err := validateCommon(patch.CommonMetadata, KindYamlPatchProposal); err != nil {
		return err
	}
	if err := validateResourceRef(patch.Resource, "YamlPatchProposal.resource"); err != nil {
		return err
	}
	if patch.PatchType == "" {
		return NewValidationError("MissingPatchType", "YamlPatchProposal.patchType is required")
	}
	if len(patch.Patch) == 0 {
		return NewValidationError("MissingPatch", "YamlPatchProposal.patch must not be empty")
	}
	if patch.Safety.DirectExecutionAllowed {
		return NewValidationError("DirectExecutionAllowed", "YamlPatchProposal.safety.directExecutionAllowed must be false")
	}
	if !patch.Safety.RequiresPR {
		return NewValidationError("PRRequired", "YamlPatchProposal.safety.requiresPR must be true")
	}
	return nil
}

func (v *BasicValidator) ValidateRiskExplanation(explanation *RiskExplanation) error {
	if explanation == nil {
		return NewValidationError("RiskExplanationNil", "risk explanation is nil")
	}
	if err := validateCommon(explanation.CommonMetadata, KindRiskExplanation); err != nil {
		return err
	}
	if explanation.PolicyResult.RiskLevel == "" {
		return NewValidationError("MissingRiskLevel", "RiskExplanation.policyResult.riskLevel is required")
	}
	if explanation.Explanation.Summary == "" {
		return NewValidationError("MissingExplanationSummary", "RiskExplanation.explanation.summary is required")
	}
	if len(explanation.Explanation.Reasons) == 0 {
		return NewValidationError("MissingExplanationReasons", "RiskExplanation.explanation.reasons must not be empty")
	}
	return nil
}

func (v *BasicValidator) ValidateRollbackPlan(plan *RollbackPlan) error {
	if plan == nil {
		return NewValidationError("RollbackPlanNil", "rollback plan is nil")
	}
	if err := validateCommon(plan.CommonMetadata, KindRollbackPlan); err != nil {
		return err
	}
	if err := validateResourceRef(plan.Target, "RollbackPlan.target"); err != nil {
		return err
	}
	if plan.RollbackType == "" {
		return NewValidationError("MissingRollbackType", "RollbackPlan.rollbackType is required")
	}
	if plan.Summary == "" {
		return NewValidationError("MissingRollbackSummary", "RollbackPlan.summary is required")
	}
	if len(plan.Steps) == 0 {
		return NewValidationError("MissingRollbackSteps", "RollbackPlan.steps must not be empty")
	}
	for i, step := range plan.Steps {
		if step.Order <= 0 {
			return NewValidationError("InvalidRollbackStepOrder", fmt.Sprintf("RollbackPlan.steps[%d].order must be > 0", i))
		}
		if step.Action == "" {
			return NewValidationError("MissingRollbackStepAction", fmt.Sprintf("RollbackPlan.steps[%d].action is required", i))
		}
	}
	if len(plan.Validation.Expected) == 0 {
		return NewValidationError("MissingRollbackValidation", "RollbackPlan.validation.expected must not be empty")
	}
	return nil
}

func (v *BasicValidator) ValidateValidationReport(report *ValidationReport) error {
	if report == nil {
		return NewValidationError("ValidationReportNil", "validation report is nil")
	}
	if err := validateCommon(report.CommonMetadata, KindValidationReport); err != nil {
		return err
	}
	if err := validateResourceRef(report.OperationRef, "ValidationReport.operationRef"); err != nil {
		return err
	}
	if err := validateResourceRef(report.Target, "ValidationReport.target"); err != nil {
		return err
	}
	if report.Result == "" {
		return NewValidationError("MissingValidationResult", "ValidationReport.result is required")
	}
	if report.Summary == "" {
		return NewValidationError("MissingValidationSummary", "ValidationReport.summary is required")
	}
	if len(report.Evidence) == 0 {
		return NewValidationError("MissingEvidence", "ValidationReport.evidence must not be empty")
	}
	for i, evidence := range report.Evidence {
		if evidence.Source == "" {
			return NewValidationError("MissingEvidenceSource", fmt.Sprintf("ValidationReport.evidence[%d].source is required", i))
		}
		if evidence.Value == "" {
			return NewValidationError("MissingEvidenceValue", fmt.Sprintf("ValidationReport.evidence[%d].value is required", i))
		}
	}
	return nil
}

func validateCommon(meta CommonMetadata, expectedKind string) error {
	if meta.SchemaVersion != SchemaVersionV1Alpha1 {
		return NewValidationError("InvalidSchemaVersion", fmt.Sprintf("schemaVersion must be %s", SchemaVersionV1Alpha1))
	}
	if meta.Kind != expectedKind {
		return NewValidationError("InvalidKind", fmt.Sprintf("kind must be %s", expectedKind))
	}
	if meta.RequestID == "" {
		return NewValidationError("MissingRequestID", "requestId is required")
	}
	if meta.TaskType == "" {
		return NewValidationError("MissingTaskType", "taskType is required")
	}
	if meta.CreatedBy == "" {
		return NewValidationError("MissingCreatedBy", "createdBy is required")
	}
	return nil
}

func validateResourceRef(ref ResourceRef, field string) error {
	if ref.Kind == "" {
		return NewValidationError("MissingResourceKind", field+".kind is required")
	}
	if ref.Name == "" {
		return NewValidationError("MissingResourceName", field+".name is required")
	}
	return nil
}

// ValidationError represents a schema validation failure.
type ValidationError struct {
	Code    string
	Message string
}

func NewValidationError(code string, message string) *ValidationError {
	return &ValidationError{Code: code, Message: message}
}

func (e *ValidationError) Error() string {
	return e.Code + ": " + e.Message
}
