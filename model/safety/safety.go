package safety

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// Validator is the model-layer safety gate.
type Validator struct {
	sensitiveKeywords     []string
	restrictedPhrases     []string
	forbiddenPatchFields  []string
	editableFieldAllowlist []string
}

func NewValidator() *Validator {
	return &Validator{
		sensitiveKeywords: []string{
			"password",
			"passwd",
			"token",
			"secret",
			"apikey",
			"api_key",
			"accesskey",
			"privatekey",
			"clientsecret",
			"authorization",
			"cookie",
			"kubeconfig",
			"credential",
		},
		restrictedPhrases: []string{
			"direct apply",
			"direct delete",
			"shell access",
			"machine power operation",
			"credential value",
			"print credential",
			"bypass policy",
			"bypass approval",
			"auto merge",
			"auto-merge",
			"auto approve",
			"auto-approve",
		},
		forbiddenPatchFields: []string{
			"status",
			"metadata.finalizers",
			"metadata.ownerReferences",
			"spec.bmcSecretRef",
			"spec.credentials",
			"spec.secretRef",
		},
		editableFieldAllowlist: []string{
			"ManagedCluster.spec.workers[].replicas",
			"spec.workers[name=gpu-workers].replicas",
		},
	}
}

// ValidateRequest blocks restricted instructions and sensitive context before provider calls.
func (v *Validator) ValidateRequest(req provider.ProviderRequest) error {
	if isUnsupportedModelTask(req.TaskType) {
		return NewSafetyError("UnsupportedModelTask", "Block", fmt.Sprintf("task %s is not supported by model providers", req.TaskType))
	}

	if v.containsRestrictedPhrase(req.Instruction) {
		return NewSafetyError("RestrictedInstruction", "Block", "request contains restricted instruction")
	}

	if v.contextContainsSensitiveKey(req.Context) {
		return NewSafetyError("SensitiveContext", "Block", "request context appears to contain sensitive fields")
	}

	return nil
}

// ValidateResponse blocks unsafe structured or raw provider output.
func (v *Validator) ValidateResponse(resp *provider.ProviderResponse) error {
	if resp == nil {
		return NewSafetyError("EmptyProviderResponse", "Block", "provider response is nil")
	}

	if v.containsRestrictedPhrase(resp.RawText) {
		return NewSafetyError("RestrictedOutput", "Block", "provider raw output contains restricted instruction")
	}

	switch structured := resp.Structured.(type) {
	case schema.ChangePlan:
		return v.validateChangePlan(&structured)
	case *schema.ChangePlan:
		return v.validateChangePlan(structured)
	case schema.YamlPatchProposal:
		return v.validateYamlPatchProposal(&structured)
	case *schema.YamlPatchProposal:
		return v.validateYamlPatchProposal(structured)
	case schema.RollbackPlan:
		return v.validateRollbackPlan(&structured)
	case *schema.RollbackPlan:
		return v.validateRollbackPlan(structured)
	case schema.ValidationReport:
		return v.validateValidationReport(&structured)
	case *schema.ValidationReport:
		return v.validateValidationReport(structured)
	case schema.RiskExplanation:
		return v.validateRiskExplanation(&structured)
	case *schema.RiskExplanation:
		return v.validateRiskExplanation(structured)
	case nil:
		return NewSafetyError("MissingStructuredOutput", "Block", "provider response does not contain structured output")
	default:
		return nil
	}
}

func (v *Validator) validateChangePlan(plan *schema.ChangePlan) error {
	if plan == nil {
		return NewSafetyError("EmptyChangePlan", "Block", "change plan is nil")
	}
	if plan.Target.Kind == "Secret" {
		return NewSafetyError("SensitiveTarget", "Block", "model output cannot target sensitive resource kind")
	}
	for _, change := range plan.Changes {
		if !v.isEditableFieldAllowed(change.Field) {
			return NewSafetyError("FieldOutsideAllowlist", "Block", "change plan modifies a field outside allowlist: "+change.Field)
		}
	}
	return nil
}

func (v *Validator) validateYamlPatchProposal(patch *schema.YamlPatchProposal) error {
	if patch == nil {
		return NewSafetyError("EmptyPatchProposal", "Block", "patch proposal is nil")
	}
	if patch.Safety.DirectExecutionAllowed {
		return NewSafetyError("DirectExecutionAllowed", "Block", "patch proposal must not allow direct execution")
	}
	if !patch.Safety.RequiresPR {
		return NewSafetyError("ReviewRequired", "Block", "patch proposal must require reviewed workflow")
	}
	if patch.Resource.Kind == "Secret" {
		return NewSafetyError("SensitivePatchTarget", "Block", "model output cannot patch sensitive resource kind")
	}
	if v.mapContainsForbiddenField(patch.Patch, "") {
		return NewSafetyError("ForbiddenFieldMutation", "Block", "patch modifies forbidden fields")
	}
	if v.mapContainsSensitiveKey(patch.Patch) {
		return NewSafetyError("SensitivePatchContent", "Block", "patch contains sensitive-looking keys or values")
	}
	return nil
}

func (v *Validator) validateRollbackPlan(plan *schema.RollbackPlan) error {
	if plan == nil {
		return NewSafetyError("EmptyRollbackPlan", "Block", "rollback plan is nil")
	}
	for _, step := range plan.Steps {
		if v.containsRestrictedPhrase(step.Action) {
			return NewSafetyError("RestrictedRollbackStep", "Block", "rollback step contains restricted instruction")
		}
	}
	if v.mapContainsForbiddenField(plan.Patch, "") {
		return NewSafetyError("ForbiddenFieldMutation", "Block", "rollback patch modifies forbidden fields")
	}
	return nil
}

func (v *Validator) validateValidationReport(report *schema.ValidationReport) error {
	if report == nil {
		return NewSafetyError("EmptyValidationReport", "Block", "validation report is nil")
	}
	if report.Target.Kind == "Secret" {
		return NewSafetyError("SensitiveValidationTarget", "Block", "validation report cannot expose sensitive resource target")
	}
	if len(report.Evidence) == 0 {
		return NewSafetyError("MissingEvidence", "Block", "validation report must include evidence")
	}
	return nil
}

func (v *Validator) validateRiskExplanation(explanation *schema.RiskExplanation) error {
	if explanation == nil {
		return NewSafetyError("EmptyRiskExplanation", "Block", "risk explanation is nil")
	}
	if explanation.PolicyResult.RiskLevel == "" {
		return NewSafetyError("MissingPolicyResult", "Block", "risk explanation must be grounded in policy result")
	}
	return nil
}

func (v *Validator) contextContainsSensitiveKey(ctx provider.ModelContext) bool {
	if v.stringContainsSensitiveKey(ctx.UserIntent) || v.stringContainsSensitiveKey(ctx.GitDiffSummary) {
		return true
	}
	for _, snippet := range ctx.RunbookSnippets {
		if v.stringContainsSensitiveKey(snippet) {
			return true
		}
	}
	for _, snapshot := range ctx.ResourceSnapshots {
		if v.mapContainsSensitiveKey(snapshot.Spec) || v.mapContainsSensitiveKey(snapshot.Status) {
			return true
		}
	}
	return false
}

func (v *Validator) mapContainsSensitiveKey(m map[string]any) bool {
	for key, value := range m {
		if v.stringContainsSensitiveKey(key) {
			return true
		}
		if nested, ok := value.(map[string]any); ok {
			if v.mapContainsSensitiveKey(nested) {
				return true
			}
		}
		if v.valueContainsSensitiveKey(value) {
			return true
		}
	}
	return false
}

func (v *Validator) valueContainsSensitiveKey(value any) bool {
	if value == nil {
		return false
	}
	if s, ok := value.(string); ok {
		return v.stringContainsSensitiveKey(s)
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			if v.valueContainsSensitiveKey(rv.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}

func (v *Validator) stringContainsSensitiveKey(s string) bool {
	lower := strings.ToLower(s)
	for _, keyword := range v.sensitiveKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func (v *Validator) containsRestrictedPhrase(text string) bool {
	lower := strings.ToLower(text)
	for _, phrase := range v.restrictedPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func (v *Validator) mapContainsForbiddenField(m map[string]any, prefix string) bool {
	for key, value := range m {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if v.isForbiddenField(path) {
			return true
		}
		if nested, ok := value.(map[string]any); ok {
			if v.mapContainsForbiddenField(nested, path) {
				return true
			}
		}
	}
	return false
}

func (v *Validator) isForbiddenField(path string) bool {
	for _, forbidden := range v.forbiddenPatchFields {
		if path == forbidden || strings.HasPrefix(path, forbidden+".") {
			return true
		}
	}
	return false
}

func (v *Validator) isEditableFieldAllowed(field string) bool {
	for _, allowed := range v.editableFieldAllowlist {
		if field == allowed {
			return true
		}
	}
	return false
}

func isUnsupportedModelTask(task provider.TaskType) bool {
	switch task {
	case provider.TaskGeneratePlan,
		provider.TaskGeneratePatch,
		provider.TaskExplainRisk,
		provider.TaskGenerateRollback,
		provider.TaskGenerateValidationReport,
		provider.TaskSummarizeState,
		provider.TaskRepairYAML,
		provider.TaskExplainPolicyFailure:
		return false
	default:
		return true
	}
}

// SafetyError is a normalized safety failure.
type SafetyError struct {
	Type    string
	Level   string
	Message string
}

func NewSafetyError(t string, level string, message string) *SafetyError {
	return &SafetyError{Type: t, Level: level, Message: message}
}

func (e *SafetyError) Error() string {
	return e.Type + ": " + e.Message
}

// RedactString is a simple placeholder redactor for logs and future audit records.
func RedactString(input string) string {
	redacted := input
	keywords := []string{"password", "token", "secret", "apikey", "privatekey", "kubeconfig", "credential"}
	for _, keyword := range keywords {
		redacted = strings.ReplaceAll(redacted, keyword, "<REDACTED_KEY>")
		redacted = strings.ReplaceAll(redacted, strings.ToUpper(keyword), "<REDACTED_KEY>")
	}
	return redacted
}
