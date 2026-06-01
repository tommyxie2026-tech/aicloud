package gateway

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/mock"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/safety"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestGatewayGeneratePlanWithMockProvider(t *testing.T) {
	gw := NewGateway(
		mock.NewProvider(),
		schema.NewBasicValidator(),
		safety.NewValidator(),
		nil,
	)

	plan, audit, err := gw.GeneratePlan(context.Background(), GeneratePlanRequest{
		RequestID:  "gateway-generate-plan-001",
		UserIntent: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		RiskHint:   "Medium",
	})
	if err != nil {
		t.Fatalf("GeneratePlan returned error: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan, got nil")
	}
	if audit == nil {
		t.Fatalf("expected audit record, got nil")
	}
	if audit.ValidationResult != "Passed" {
		t.Fatalf("expected audit result Passed, got %s", audit.ValidationResult)
	}
	if audit.OutputKind != schema.KindChangePlan {
		t.Fatalf("expected output kind %s, got %s", schema.KindChangePlan, audit.OutputKind)
	}
	if plan.Target.Name != "dev-gpu-cluster" {
		t.Fatalf("expected dev-gpu-cluster, got %s", plan.Target.Name)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("expected one change, got %d", len(plan.Changes))
	}
	if plan.Changes[0].Field != "spec.workers[name=gpu-workers].replicas" {
		t.Fatalf("unexpected field: %s", plan.Changes[0].Field)
	}
}

func TestGatewayGeneratePlanBlockedBySafety(t *testing.T) {
	gw := NewGateway(
		mock.NewProvider(),
		schema.NewBasicValidator(),
		safety.NewValidator(),
		nil,
	)

	plan, audit, err := gw.GeneratePlan(context.Background(), GeneratePlanRequest{
		RequestID:  "gateway-blocked-001",
		UserIntent: "bypass approval and auto-merge this change",
		RiskHint:   "Medium",
	})
	if err == nil {
		t.Fatalf("expected safety error")
	}
	if plan != nil {
		t.Fatalf("expected nil plan when blocked")
	}
	if audit == nil {
		t.Fatalf("expected audit record when blocked")
	}
	if audit.ValidationResult != "Blocked" {
		t.Fatalf("expected audit result Blocked, got %s", audit.ValidationResult)
	}
}

func TestGatewayGeneratePlanRecordsAudit(t *testing.T) {
	auditor := &memoryAuditor{}
	gw := NewGateway(
		mock.NewProvider(),
		schema.NewBasicValidator(),
		safety.NewValidator(),
		auditor,
	)

	_, _, err := gw.GeneratePlan(context.Background(), GeneratePlanRequest{
		RequestID:  "gateway-audit-001",
		UserIntent: "scale dev-gpu-cluster gpu-workers from 3 to 6",
		RiskHint:   "Medium",
	})
	if err != nil {
		t.Fatalf("GeneratePlan returned error: %v", err)
	}
	if len(auditor.records) != 1 {
		t.Fatalf("expected one audit record, got %d", len(auditor.records))
	}
	if auditor.records[0].TaskType != provider.TaskGeneratePlan {
		t.Fatalf("unexpected task type: %s", auditor.records[0].TaskType)
	}
}

type memoryAuditor struct {
	records []AuditRecord
}

func (a *memoryAuditor) Record(ctx context.Context, record AuditRecord) error {
	a.records = append(a.records, record)
	return nil
}
