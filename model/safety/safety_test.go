package safety

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestValidateRequestAllowsSafeRequest(t *testing.T) {
	v := NewValidator()

	err := v.ValidateRequest(provider.ProviderRequest{
		RequestID:   "safe-request-001",
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "scale dev-gpu-cluster gpu-workers from 3 to 6",
	})
	if err != nil {
		t.Fatalf("expected safe request to pass, got error: %v", err)
	}
}

func TestValidateRequestBlocksRestrictedInstruction(t *testing.T) {
	v := NewValidator()

	err := v.ValidateRequest(provider.ProviderRequest{
		RequestID:   "blocked-request-001",
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "bypass approval and auto-merge this change",
	})
	if err == nil {
		t.Fatalf("expected restricted instruction to be blocked")
	}
}

func TestValidateRequestBlocksSensitiveContext(t *testing.T) {
	v := NewValidator()

	err := v.ValidateRequest(provider.ProviderRequest{
		RequestID:   "sensitive-context-001",
		TaskType:    provider.TaskGeneratePlan,
		Instruction: "summarize this context",
		Context: provider.ModelContext{
			ResourceSnapshots: []provider.SanitizedResourceSnapshot{
				{
					Ref: provider.ResourceRef{Kind: "Config", Name: "example"},
					Spec: map[string]any{
						"apiKey": "example-value",
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatalf("expected sensitive context to be blocked")
	}
}

func TestValidateResponseAllowsSafeChangePlan(t *testing.T) {
	v := NewValidator()

	plan := schema.ChangePlan{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindChangePlan,
			RequestID:     "safe-plan-001",
			TaskType:      string(provider.TaskGeneratePlan),
			CreatedBy:     "test",
		},
		Intent:        "scale dev-gpu-cluster gpu-workers from 3 to 6",
		Target:        schema.ResourceRef{Kind: "ManagedCluster", Name: "dev-gpu-cluster"},
		OperationType: "ScaleOut",
		Changes: []schema.PlannedChange{
			{Field: "spec.workers[name=gpu-workers].replicas", From: 3, To: 6},
		},
		Rollback: schema.RollbackSummary{Summary: "set replicas back to 3"},
	}

	err := v.ValidateResponse(&provider.ProviderResponse{
		RequestID:   "safe-plan-001",
		TaskType:    provider.TaskGeneratePlan,
		Structured: plan,
	})
	if err != nil {
		t.Fatalf("expected safe change plan to pass, got error: %v", err)
	}
}

func TestValidateResponseBlocksFieldOutsideAllowlist(t *testing.T) {
	v := NewValidator()

	plan := schema.ChangePlan{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindChangePlan,
			RequestID:     "unsafe-plan-001",
			TaskType:      string(provider.TaskGeneratePlan),
			CreatedBy:     "test",
		},
		Intent:        "change network topology",
		Target:        schema.ResourceRef{Kind: "ManagedCluster", Name: "dev-gpu-cluster"},
		OperationType: "NetworkChange",
		Changes: []schema.PlannedChange{
			{Field: "spec.network.cidr", From: "10.0.0.0/16", To: "10.1.0.0/16"},
		},
		Rollback: schema.RollbackSummary{Summary: "restore previous network cidr"},
	}

	err := v.ValidateResponse(&provider.ProviderResponse{
		RequestID:   "unsafe-plan-001",
		TaskType:    provider.TaskGeneratePlan,
		Structured: plan,
	})
	if err == nil {
		t.Fatalf("expected field outside allowlist to be blocked")
	}
}

func TestValidateResponseBlocksValidationReportWithoutEvidence(t *testing.T) {
	v := NewValidator()

	report := schema.ValidationReport{
		CommonMetadata: schema.CommonMetadata{
			SchemaVersion: schema.SchemaVersionV1Alpha1,
			Kind:          schema.KindValidationReport,
			RequestID:     "report-001",
			TaskType:      string(provider.TaskGenerateValidationReport),
			CreatedBy:     "test",
		},
		OperationRef: schema.ResourceRef{Kind: "AgentOperation", Name: "op-001"},
		Target:       schema.ResourceRef{Kind: "ManagedCluster", Name: "dev-gpu-cluster"},
		Result:       "Succeeded",
		Summary:      "completed",
		Evidence:     nil,
	}

	err := v.ValidateResponse(&provider.ProviderResponse{
		RequestID:   "report-001",
		TaskType:    provider.TaskGenerateValidationReport,
		Structured: report,
	})
	if err == nil {
		t.Fatalf("expected validation report without evidence to be blocked")
	}
}
