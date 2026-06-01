package provider

import "context"

// ModelProvider is the common interface for all model backends.
//
// Providers generate structured outputs only. They must not expose direct
// infrastructure execution capabilities such as cluster writes, shell access,
// credential reads, automatic approval, or automatic merge.
type ModelProvider interface {
	Name() string
	Type() ProviderType
	Capabilities() ProviderCapabilities
	Generate(ctx context.Context, req ProviderRequest) (*ProviderResponse, error)
	Health(ctx context.Context) (*ProviderHealth, error)
}

// ProviderType describes the deployment and ownership model of a provider.
type ProviderType string

const (
	ProviderTypeHosted       ProviderType = "Hosted"
	ProviderTypePrivate      ProviderType = "Private"
	ProviderTypeLocal        ProviderType = "Local"
	ProviderTypeMock         ProviderType = "Mock"
	ProviderTypeCustomDomain ProviderType = "CustomDomain"
)

// TaskType describes a safe model task.
type TaskType string

const (
	TaskGeneratePlan             TaskType = "GeneratePlan"
	TaskGeneratePatch            TaskType = "GeneratePatch"
	TaskExplainRisk              TaskType = "ExplainRisk"
	TaskGenerateRollback         TaskType = "GenerateRollback"
	TaskGenerateValidationReport TaskType = "GenerateValidationReport"
	TaskSummarizeState           TaskType = "SummarizeState"
	TaskRepairYAML               TaskType = "RepairYAML"
	TaskExplainPolicyFailure     TaskType = "ExplainPolicyFailure"
)

// Restricted capability names are tracked as strings for policy and tests.
// They are intentionally not methods on ModelProvider.
const (
	RestrictedDirectExecution = "DirectExecution"
	RestrictedManifestApply   = "ManifestApply"
	RestrictedCredentialRead  = "CredentialRead"
	RestrictedMachineControl  = "MachineControl"
	RestrictedProductionDelete = "ProductionDelete"
	RestrictedAutoApprove     = "AutoApprove"
	RestrictedAutoMerge       = "AutoMerge"
)

// ProviderCapabilities describes what a provider can safely support.
type ProviderCapabilities struct {
	SupportsStructuredOutput bool       `json:"supportsStructuredOutput"`
	SupportsJSONSchema      bool       `json:"supportsJSONSchema"`
	SupportsStreaming       bool       `json:"supportsStreaming"`
	SupportsToolUse         bool       `json:"supportsToolUse"`
	SupportsVision          bool       `json:"supportsVision"`
	SupportsLongContext     bool       `json:"supportsLongContext"`
	SupportsChinese         bool       `json:"supportsChinese"`
	SupportsCodeGeneration  bool       `json:"supportsCodeGeneration"`
	SupportsLocalDeployment bool       `json:"supportsLocalDeployment"`
	MaxInputTokens          int        `json:"maxInputTokens"`
	MaxOutputTokens         int        `json:"maxOutputTokens"`
	RecommendedTasks        []TaskType `json:"recommendedTasks,omitempty"`
	RestrictedCapabilities  []string   `json:"restrictedCapabilities,omitempty"`
}

// ProviderRequest is the normalized request sent from the gateway to a provider.
type ProviderRequest struct {
	RequestID       string          `json:"requestId"`
	TaskType        TaskType        `json:"taskType"`
	UserID          string          `json:"userId,omitempty"`
	RiskHint        string          `json:"riskHint,omitempty"`
	SystemPrompt    string          `json:"systemPrompt,omitempty"`
	Instruction     string          `json:"instruction"`
	Context         ModelContext    `json:"context,omitempty"`
	OutputSchema    OutputSchemaRef `json:"outputSchema"`
	SafetyPolicy    SafetyPolicyRef `json:"safetyPolicy,omitempty"`
	MaxOutputTokens int             `json:"maxOutputTokens,omitempty"`
	Temperature     float32         `json:"temperature,omitempty"`
}

// ModelContext is a sanitized, bounded context object.
type ModelContext struct {
	ResourceRefs      []ResourceRef               `json:"resourceRefs,omitempty"`
	ResourceSnapshots []SanitizedResourceSnapshot `json:"resourceSnapshots,omitempty"`
	PolicyResult      *PolicyResultSnapshot       `json:"policyResult,omitempty"`
	GitDiffSummary    string                      `json:"gitDiffSummary,omitempty"`
	RunbookSnippets   []string                    `json:"runbookSnippets,omitempty"`
	PriorPRSummaries  []string                    `json:"priorPrSummaries,omitempty"`
	UserIntent        string                      `json:"userIntent,omitempty"`
}

type ResourceRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type SanitizedResourceSnapshot struct {
	Ref        ResourceRef        `json:"ref"`
	Spec       map[string]any     `json:"spec,omitempty"`
	Status     map[string]any     `json:"status,omitempty"`
	Conditions []ConditionSummary `json:"conditions,omitempty"`
}

type ConditionSummary struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type PolicyResultSnapshot struct {
	Result           string `json:"result"`
	RiskLevel        string `json:"riskLevel"`
	ApprovalRequired bool   `json:"approvalRequired"`
	PolicyName       string `json:"policyName,omitempty"`
	MatchedRule      string `json:"matchedRule,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

type OutputSchemaRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SafetyPolicyRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ProviderResponse is the normalized response returned by a provider.
type ProviderResponse struct {
	RequestID       string         `json:"requestId"`
	ProviderName    string         `json:"providerName"`
	ModelName       string         `json:"modelName"`
	TaskType        TaskType       `json:"taskType"`
	RawText         string         `json:"rawText,omitempty"`
	Structured      any            `json:"structured,omitempty"`
	FinishReason    string         `json:"finishReason,omitempty"`
	TokenUsage      TokenUsage     `json:"tokenUsage,omitempty"`
	LatencyMs       int64          `json:"latencyMs,omitempty"`
	SafetySignals   []SafetySignal `json:"safetySignals,omitempty"`
	ValidationHints []string       `json:"validationHints,omitempty"`
}

type TokenUsage struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

type SafetySignal struct {
	Type    string `json:"type"`
	Level   string `json:"level"`
	Message string `json:"message,omitempty"`
}

// ProviderHealth describes provider availability and basic health.
type ProviderHealth struct {
	Name       string   `json:"name"`
	Available  bool     `json:"available"`
	LatencyMs  int64    `json:"latencyMs,omitempty"`
	ModelNames []string `json:"modelNames,omitempty"`
	Message    string   `json:"message,omitempty"`
}

// ProviderErrorCode normalizes model-provider failures.
type ProviderErrorCode string

const (
	ErrProviderUnavailable ProviderErrorCode = "ProviderUnavailable"
	ErrProviderTimeout     ProviderErrorCode = "ProviderTimeout"
	ErrRateLimited         ProviderErrorCode = "RateLimited"
	ErrInvalidOutput       ProviderErrorCode = "InvalidOutput"
	ErrSchemaMismatch      ProviderErrorCode = "SchemaMismatch"
	ErrUnsafeOutput        ProviderErrorCode = "UnsafeOutput"
	ErrContextTooLarge     ProviderErrorCode = "ContextTooLarge"
	ErrCostLimitExceeded   ProviderErrorCode = "CostLimitExceeded"
)

type ProviderError struct {
	Code      ProviderErrorCode `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable"`
}

func (e *ProviderError) Error() string {
	return string(e.Code) + ": " + e.Message
}
