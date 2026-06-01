package mock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// Provider is a deterministic provider for tests and offline development.
type Provider struct {
	name string
}

func NewProvider() *Provider {
	return &Provider{name: "mock"}
}

func (p *Provider) Name() string {
	return p.name
}

func (p *Provider) Type() provider.ProviderType {
	return provider.ProviderTypeMock
}

func (p *Provider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		SupportsStructuredOutput: true,
		SupportsJSONSchema:      true,
		SupportsStreaming:       false,
		SupportsToolUse:         false,
		SupportsVision:          false,
		SupportsLongContext:     false,
		SupportsChinese:         true,
		SupportsCodeGeneration:  false,
		SupportsLocalDeployment: true,
		MaxInputTokens:          4096,
		MaxOutputTokens:         2048,
		RecommendedTasks: []provider.TaskType{
			provider.TaskGeneratePlan,
			provider.TaskGenerateRollback,
			provider.TaskGenerateValidationReport,
			provider.TaskExplainRisk,
		},
		RestrictedCapabilities: []string{
			provider.RestrictedDirectExecution,
			provider.RestrictedManifestApply,
			provider.RestrictedCredentialRead,
			provider.RestrictedMachineControl,
			provider.RestrictedProductionDelete,
			provider.RestrictedAutoApprove,
			provider.RestrictedAutoMerge,
		},
	}
}

func (p *Provider) Health(ctx context.Context) (*provider.ProviderHealth, error) {
	return &provider.ProviderHealth{
		Name:       p.name,
		Available:  true,
		LatencyMs:  1,
		ModelNames: []string{"mock-deterministic-v1"},
		Message:    "mock provider is available",
	}, nil
}

func (p *Provider) Generate(ctx context.Context, req provider.ProviderRequest) (*provider.ProviderResponse, error) {
	startedAt := time.Now()

	if restrictedInstruction(req.Instruction) {
		return &provider.ProviderResponse{
			RequestID:    req.RequestID,
			ProviderName: p.name,
			ModelName:    "mock-deterministic-v1",
			TaskType:     req.TaskType,
			FinishReason: "blocked",
			LatencyMs:    time.Since(startedAt).Milliseconds(),
			SafetySignals: []provider.SafetySignal{
				{Type: "RestrictedInstruction", Level: "Block", Message: "mock provider blocked restricted instruction"},
			},
		}, &provider.ProviderError{Code: provider.ErrUnsafeOutput, Message: "restricted instruction blocked by mock provider", Retryable: false}
	}

	switch req.TaskType {
	case provider.TaskGeneratePlan:
		return p.generatePlan(req, startedAt)
	case provider.TaskGenerateRollback:
		return p.generateRollback(req, startedAt)
	case provider.TaskGenerateValidationReport:
		return p.generateValidationReport(req, startedAt)
	case provider.TaskExplainRisk:
		return p.explainRisk(req, startedAt)
	default:
		return nil, &provider.ProviderError{Code: provider.ErrInvalidOutput, Message: fmt.Sprintf("unsupported mock task type: %s", req.TaskType), Retryable: false}
	}
}

func (p *Provider) generatePlan(req provider.ProviderRequest, startedAt time.Time) (*provider.ProviderResponse, error) {
	plan := schema.ChangePlan{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindChangePlan,
			RequestID:     req.RequestID,
			TaskType:      string(provider.TaskGeneratePlan),
			CreatedBy:     "model-gateway",
			Model:         &schema.ModelRef{Provider: p.name, Name: "mock-deterministic-v1"},
			Confidence:    &schema.ConfidenceHint{Level: "High", Notes: []string{"deterministic fixture for dev scale-out"}},
		},
		Intent: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		Target: schema.ResourceRef{
			APIVersion: "infra.ai/v1alpha1",
			Kind:       "ManagedCluster",
			Namespace:  "default",
			Name:       "dev-gpu-cluster",
		},
		OperationType: "ScaleOut",
		Environment:   "dev",
		RiskHint:      "Medium",
		Changes: []schema.PlannedChange{
			{Field: "spec.workers[name=gpu-workers].replicas", From: 3, To: 6, Reason: "user requested dev GPU worker scale-out"},
		},
		Rollback: schema.RollbackSummary{Summary: "set gpu-workers replicas back to 3"},
		Validation: schema.ValidationExpectations{Expected: []string{
			"ManagedCluster Ready=True",
			"workerReadyReplicas=6",
		}},
	}

	return &provider.ProviderResponse{
		RequestID:    req.RequestID,
		ProviderName: p.name,
		ModelName:    "mock-deterministic-v1",
		TaskType:     req.TaskType,
		Structured:   plan,
		FinishReason: "stop",
		LatencyMs:    time.Since(startedAt).Milliseconds(),
		TokenUsage:   provider.TokenUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
	}, nil
}

func (p *Provider) generateRollback(req provider.ProviderRequest, startedAt time.Time) (*provider.ProviderResponse, error) {
	rollback := schema.RollbackPlan{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindRollbackPlan,
			RequestID:     req.RequestID,
			TaskType:      string(provider.TaskGenerateRollback),
			CreatedBy:     "model-gateway",
			Model:         &schema.ModelRef{Provider: p.name, Name: "mock-deterministic-v1"},
		},
		Target: schema.ResourceRef{
			APIVersion: "infra.ai/v1alpha1",
			Kind:       "ManagedCluster",
			Namespace:  "default",
			Name:       "dev-gpu-cluster",
		},
		OperationType: "ScaleOut",
		RollbackType:  "ReversePatch",
		Summary:       "Set gpu-workers replicas from 6 back to 3.",
		Steps: []schema.RollbackStep{
			{Order: 1, Action: "Create a reviewed change that sets replicas from 6 to 3."},
			{Order: 2, Action: "Wait for the approved delivery workflow to complete."},
			{Order: 3, Action: "Verify workerReadyReplicas returns to 3."},
		},
		Patch: map[string]any{
			"spec": map[string]any{
				"workers": []map[string]any{{
					"name":     "gpu-workers",
					"replicas": 3,
					"machineClassRef": map[string]any{
						"name": "gpu-large",
					},
				}},
			},
		},
		Validation: schema.ValidationExpectations{Expected: []string{"workerReadyReplicas=3"}},
	}

	return &provider.ProviderResponse{RequestID: req.RequestID, ProviderName: p.name, ModelName: "mock-deterministic-v1", TaskType: req.TaskType, Structured: rollback, FinishReason: "stop", LatencyMs: time.Since(startedAt).Milliseconds()}, nil
}

func (p *Provider) generateValidationReport(req provider.ProviderRequest, startedAt time.Time) (*provider.ProviderResponse, error) {
	report := schema.ValidationReport{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindValidationReport,
			RequestID:     req.RequestID,
			TaskType:      string(provider.TaskGenerateValidationReport),
			CreatedBy:     "model-gateway",
			Model:         &schema.ModelRef{Provider: p.name, Name: "mock-deterministic-v1"},
		},
		OperationRef: schema.ResourceRef{APIVersion: "infra.ai/v1alpha1", Kind: "AgentOperation", Namespace: "default", Name: "scale-dev-gpu-cluster"},
		Target:       schema.ResourceRef{APIVersion: "infra.ai/v1alpha1", Kind: "ManagedCluster", Namespace: "default", Name: "dev-gpu-cluster"},
		ObservedState: schema.ObservedState{
			Phase:           "Running",
			DesiredReplicas: 6,
			ReadyReplicas:   6,
			Conditions:      []schema.ConditionSummary{{Type: "Ready", Status: "True", Reason: "WorkersReady"}},
		},
		Result:  "Succeeded",
		Summary: "The scale-out completed successfully.",
		Evidence: []schema.EvidenceItem{
			{Source: "ManagedCluster.status.workerReadyReplicas", Value: "6"},
			{Source: "ManagedCluster.status.conditions[Ready]", Value: "True"},
		},
	}

	return &provider.ProviderResponse{RequestID: req.RequestID, ProviderName: p.name, ModelName: "mock-deterministic-v1", TaskType: req.TaskType, Structured: report, FinishReason: "stop", LatencyMs: time.Since(startedAt).Milliseconds()}, nil
}

func (p *Provider) explainRisk(req provider.ProviderRequest, startedAt time.Time) (*provider.ProviderResponse, error) {
	explanation := schema.RiskExplanation{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindRiskExplanation,
			RequestID:     req.RequestID,
			TaskType:      string(provider.TaskExplainRisk),
			CreatedBy:     "model-gateway",
			Model:         &schema.ModelRef{Provider: p.name, Name: "mock-deterministic-v1"},
		},
		PolicyResult: schema.PolicyResultRef{RiskLevel: "Medium", ApprovalRequired: false, PolicyName: "default-risk-policy", MatchedRule: "dev-worker-scale-small", Result: "PASS", Reason: "dev scale-out within small threshold"},
		Explanation: schema.Explanation{Summary: "This is a medium-risk dev scale-out.", Reasons: []string{"The target environment is dev.", "The replica increase is +3.", "A rollback plan is available."}},
		ReviewerNotes: []string{"Verify capacity before merging the reviewed change."},
	}

	return &provider.ProviderResponse{RequestID: req.RequestID, ProviderName: p.name, ModelName: "mock-deterministic-v1", TaskType: req.TaskType, Structured: explanation, FinishReason: "stop", LatencyMs: time.Since(startedAt).Milliseconds()}, nil
}

func restrictedInstruction(instruction string) bool {
	lower := strings.ToLower(instruction)
	restrictedTerms := []string{
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
	}
	for _, term := range restrictedTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}
