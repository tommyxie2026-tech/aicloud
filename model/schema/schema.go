package schema

// CommonMetadata is embedded in every model-generated structured output.
type CommonMetadata struct {
	SchemaVersion string          `json:"schemaVersion" yaml:"schemaVersion"`
	Kind          string          `json:"kind" yaml:"kind"`
	RequestID     string          `json:"requestId" yaml:"requestId"`
	TaskType      string          `json:"taskType" yaml:"taskType"`
	CreatedBy     string          `json:"createdBy" yaml:"createdBy"`
	Model         *ModelRef       `json:"model,omitempty" yaml:"model,omitempty"`
	Confidence    *ConfidenceHint `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	Warnings      []string        `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

type ModelRef struct {
	Provider string `json:"provider" yaml:"provider"`
	Name     string `json:"name" yaml:"name"`
}

type ConfidenceHint struct {
	Level string   `json:"level" yaml:"level"`
	Notes []string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// ResourceRef identifies a Kubernetes-style resource.
type ResourceRef struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string `json:"kind" yaml:"kind"`
	Namespace  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Name       string `json:"name" yaml:"name"`
}

// ChangePlan represents a model-generated plan. It is not directly executable.
type ChangePlan struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	Intent         string                 `json:"intent" yaml:"intent"`
	Target         ResourceRef            `json:"target" yaml:"target"`
	OperationType  string                 `json:"operationType" yaml:"operationType"`
	Environment    string                 `json:"environment,omitempty" yaml:"environment,omitempty"`
	RiskHint       string                 `json:"riskHint,omitempty" yaml:"riskHint,omitempty"`
	Changes        []PlannedChange        `json:"changes" yaml:"changes"`
	Rollback       RollbackSummary        `json:"rollback" yaml:"rollback"`
	Validation     ValidationExpectations `json:"validation,omitempty" yaml:"validation,omitempty"`
}

type PlannedChange struct {
	Field  string `json:"field" yaml:"field"`
	From   any    `json:"from,omitempty" yaml:"from,omitempty"`
	To     any    `json:"to" yaml:"to"`
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type RollbackSummary struct {
	Summary string `json:"summary" yaml:"summary"`
}

type ValidationExpectations struct {
	Expected []string `json:"expected,omitempty" yaml:"expected,omitempty"`
}

// YamlPatchProposal represents a model-generated patch proposal.
// It can only be used to create a reviewable change, never direct execution.
type YamlPatchProposal struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	Resource       ResourceRef    `json:"resource" yaml:"resource"`
	PatchType      string         `json:"patchType" yaml:"patchType"`
	Patch          map[string]any `json:"patch" yaml:"patch"`
	SourcePlanRef  *SourcePlanRef `json:"sourcePlanRef,omitempty" yaml:"sourcePlanRef,omitempty"`
	Safety         PatchSafety    `json:"safety" yaml:"safety"`
}

type SourcePlanRef struct {
	RequestID string `json:"requestId" yaml:"requestId"`
}

type PatchSafety struct {
	DirectExecutionAllowed bool `json:"directExecutionAllowed" yaml:"directExecutionAllowed"`
	RequiresPR             bool `json:"requiresPR" yaml:"requiresPR"`
}

// RiskExplanation explains deterministic policy output.
type RiskExplanation struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	PolicyResult   PolicyResultRef `json:"policyResult" yaml:"policyResult"`
	Explanation    Explanation     `json:"explanation" yaml:"explanation"`
	ReviewerNotes  []string        `json:"reviewerNotes,omitempty" yaml:"reviewerNotes,omitempty"`
}

type PolicyResultRef struct {
	RiskLevel        string `json:"riskLevel" yaml:"riskLevel"`
	ApprovalRequired bool   `json:"approvalRequired" yaml:"approvalRequired"`
	PolicyName       string `json:"policyName,omitempty" yaml:"policyName,omitempty"`
	MatchedRule      string `json:"matchedRule,omitempty" yaml:"matchedRule,omitempty"`
	Result           string `json:"result,omitempty" yaml:"result,omitempty"`
	Reason           string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type Explanation struct {
	Summary string   `json:"summary" yaml:"summary"`
	Reasons []string `json:"reasons,omitempty" yaml:"reasons,omitempty"`
}

// RollbackPlan describes the safe reverse path for a proposed change.
type RollbackPlan struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	Target         ResourceRef            `json:"target" yaml:"target"`
	OperationType  string                 `json:"operationType" yaml:"operationType"`
	RollbackType   string                 `json:"rollbackType" yaml:"rollbackType"`
	Summary        string                 `json:"summary" yaml:"summary"`
	Steps          []RollbackStep         `json:"steps" yaml:"steps"`
	Patch          map[string]any         `json:"patch,omitempty" yaml:"patch,omitempty"`
	Validation     ValidationExpectations `json:"validation,omitempty" yaml:"validation,omitempty"`
}

type RollbackStep struct {
	Order  int    `json:"order" yaml:"order"`
	Action string `json:"action" yaml:"action"`
}

// ValidationReport summarizes actual observed state after a change.
type ValidationReport struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	OperationRef   ResourceRef    `json:"operationRef" yaml:"operationRef"`
	Target         ResourceRef    `json:"target" yaml:"target"`
	ObservedState  ObservedState  `json:"observedState" yaml:"observedState"`
	Result         string         `json:"result" yaml:"result"`
	Summary        string         `json:"summary" yaml:"summary"`
	Evidence       []EvidenceItem `json:"evidence" yaml:"evidence"`
}

type ObservedState struct {
	Phase           string             `json:"phase,omitempty" yaml:"phase,omitempty"`
	DesiredReplicas int32              `json:"desiredReplicas,omitempty" yaml:"desiredReplicas,omitempty"`
	ReadyReplicas   int32              `json:"readyReplicas,omitempty" yaml:"readyReplicas,omitempty"`
	Conditions      []ConditionSummary `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

type ConditionSummary struct {
	Type    string `json:"type" yaml:"type"`
	Status  string `json:"status" yaml:"status"`
	Reason  string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message string `json:"message,omitempty" yaml:"message,omitempty"`
}

type EvidenceItem struct {
	Source string `json:"source" yaml:"source"`
	Value  string `json:"value" yaml:"value"`
}

// StateSummary summarizes current state for humans.
type StateSummary struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	Target         ResourceRef `json:"target" yaml:"target"`
	Summary        string      `json:"summary" yaml:"summary"`
	Observations   []string    `json:"observations,omitempty" yaml:"observations,omitempty"`
}

// PolicyFailureExplanation explains a deterministic policy failure.
type PolicyFailureExplanation struct {
	CommonMetadata `json:",inline" yaml:",inline"`
	PolicyResult   PolicyResultRef    `json:"policyResult" yaml:"policyResult"`
	Explanation    FailureExplanation `json:"explanation" yaml:"explanation"`
}

type FailureExplanation struct {
	Summary       string   `json:"summary" yaml:"summary"`
	RequiredFixes []string `json:"requiredFixes,omitempty" yaml:"requiredFixes,omitempty"`
}

// Validator validates structured model outputs.
type Validator interface {
	ValidateChangePlan(plan *ChangePlan) error
	ValidateYamlPatchProposal(patch *YamlPatchProposal) error
	ValidateRiskExplanation(explanation *RiskExplanation) error
	ValidateRollbackPlan(plan *RollbackPlan) error
	ValidateValidationReport(report *ValidationReport) error
}

const (
	SchemaVersionV1Alpha1 = "ai.infra/v1alpha1"

	KindChangePlan               = "ChangePlan"
	KindYamlPatchProposal        = "YamlPatchProposal"
	KindRiskExplanation          = "RiskExplanation"
	KindRollbackPlan             = "RollbackPlan"
	KindValidationReport         = "ValidationReport"
	KindStateSummary             = "StateSummary"
	KindPolicyFailureExplanation = "PolicyFailureExplanation"
)

var EditableFieldAllowlist = []string{
	"ManagedCluster.spec.workers[].replicas",
	"spec.workers[name=gpu-workers].replicas",
}

var ForbiddenPatchFields = []string{
	"status",
	"metadata.finalizers",
	"metadata.ownerReferences",
	"spec.bmcSecretRef",
	"spec.credentials",
	"spec.secretRef",
}
