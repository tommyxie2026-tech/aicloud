package openai

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestJSONStructuredParserParsesChangePlan(t *testing.T) {
	parser := NewJSONStructuredParser()

	out, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindChangePlan, Version: schema.SchemaVersionV1Alpha1}, validChangePlanJSON())
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	plan, ok := out.(schema.ChangePlan)
	if !ok {
		t.Fatalf("expected ChangePlan, got %T", out)
	}
	if plan.Target.Name != "dev-gpu-cluster" {
		t.Fatalf("expected dev-gpu-cluster, got %s", plan.Target.Name)
	}
}

func TestJSONStructuredParserParsesRollbackPlan(t *testing.T) {
	parser := NewJSONStructuredParser()

	out, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindRollbackPlan, Version: schema.SchemaVersionV1Alpha1}, `{
		"schemaVersion":"ai.infra/v1alpha1",
		"kind":"RollbackPlan",
		"requestId":"rollback-001",
		"taskType":"GenerateRollback",
		"createdBy":"model-gateway",
		"target":{"kind":"ManagedCluster","name":"dev-gpu-cluster"},
		"operationType":"ScaleOut",
		"rollbackType":"ReversePatch",
		"summary":"set replicas back to 3",
		"steps":[{"order":1,"action":"create reviewed rollback change"}],
		"validation":{"expected":["workerReadyReplicas=3"]}
	}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	plan, ok := out.(schema.RollbackPlan)
	if !ok {
		t.Fatalf("expected RollbackPlan, got %T", out)
	}
	if plan.RollbackType != "ReversePatch" {
		t.Fatalf("expected ReversePatch, got %s", plan.RollbackType)
	}
}

func TestJSONStructuredParserParsesValidationReport(t *testing.T) {
	parser := NewJSONStructuredParser()

	out, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindValidationReport, Version: schema.SchemaVersionV1Alpha1}, `{
		"schemaVersion":"ai.infra/v1alpha1",
		"kind":"ValidationReport",
		"requestId":"report-001",
		"taskType":"GenerateValidationReport",
		"createdBy":"model-gateway",
		"operationRef":{"kind":"AgentOperation","name":"op-001"},
		"target":{"kind":"ManagedCluster","name":"dev-gpu-cluster"},
		"observedState":{"phase":"Running","desiredReplicas":6,"readyReplicas":6},
		"result":"Succeeded",
		"summary":"completed",
		"evidence":[{"source":"status.readyReplicas","value":"6"}]
	}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	report, ok := out.(schema.ValidationReport)
	if !ok {
		t.Fatalf("expected ValidationReport, got %T", out)
	}
	if report.Result != "Succeeded" {
		t.Fatalf("expected Succeeded, got %s", report.Result)
	}
}

func TestJSONStructuredParserParsesRiskExplanation(t *testing.T) {
	parser := NewJSONStructuredParser()

	out, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindRiskExplanation, Version: schema.SchemaVersionV1Alpha1}, `{
		"schemaVersion":"ai.infra/v1alpha1",
		"kind":"RiskExplanation",
		"requestId":"risk-001",
		"taskType":"ExplainRisk",
		"createdBy":"model-gateway",
		"policyResult":{"riskLevel":"Medium","approvalRequired":false,"policyName":"default"},
		"explanation":{"summary":"medium risk","reasons":["dev environment"]}
	}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	explanation, ok := out.(schema.RiskExplanation)
	if !ok {
		t.Fatalf("expected RiskExplanation, got %T", out)
	}
	if explanation.PolicyResult.RiskLevel != "Medium" {
		t.Fatalf("expected Medium, got %s", explanation.PolicyResult.RiskLevel)
	}
}

func TestJSONStructuredParserRejectsUnknownField(t *testing.T) {
	parser := NewJSONStructuredParser()

	_, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindChangePlan, Version: schema.SchemaVersionV1Alpha1}, `{
		"schemaVersion":"ai.infra/v1alpha1",
		"kind":"ChangePlan",
		"requestId":"plan-001",
		"taskType":"GeneratePlan",
		"createdBy":"model-gateway",
		"intent":"scale",
		"target":{"kind":"ManagedCluster","name":"dev-gpu-cluster"},
		"operationType":"ScaleOut",
		"changes":[{"field":"spec.workers[name=gpu-workers].replicas","to":6}],
		"rollback":{"summary":"back to 3"},
		"unexpectedField":"must fail"
	}`)
	if err == nil {
		t.Fatalf("expected unknown field to be rejected")
	}
}

func TestJSONStructuredParserRejectsTrailingJSON(t *testing.T) {
	parser := NewJSONStructuredParser()

	_, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindChangePlan, Version: schema.SchemaVersionV1Alpha1}, validChangePlanJSON()+` {"another":"object"}`)
	if err == nil {
		t.Fatalf("expected trailing JSON to be rejected")
	}
}

func TestJSONStructuredParserRejectsEmptyOutput(t *testing.T) {
	parser := NewJSONStructuredParser()

	_, err := parser.Parse(provider.OutputSchemaRef{Name: schema.KindChangePlan, Version: schema.SchemaVersionV1Alpha1}, "   ")
	if err == nil {
		t.Fatalf("expected empty output to be rejected")
	}
}

func TestJSONStructuredParserRejectsUnsupportedSchema(t *testing.T) {
	parser := NewJSONStructuredParser()

	_, err := parser.Parse(provider.OutputSchemaRef{Name: "UnknownKind", Version: schema.SchemaVersionV1Alpha1}, `{}`)
	if err == nil {
		t.Fatalf("expected unsupported schema to be rejected")
	}
}

func validChangePlanJSON() string {
	return `{
		"schemaVersion":"ai.infra/v1alpha1",
		"kind":"ChangePlan",
		"requestId":"plan-001",
		"taskType":"GeneratePlan",
		"createdBy":"model-gateway",
		"intent":"scale dev-gpu-cluster gpu-workers from 3 to 6",
		"target":{"apiVersion":"infra.ai/v1alpha1","kind":"ManagedCluster","namespace":"default","name":"dev-gpu-cluster"},
		"operationType":"ScaleOut",
		"environment":"dev",
		"riskHint":"Medium",
		"changes":[{"field":"spec.workers[name=gpu-workers].replicas","from":3,"to":6,"reason":"scale out"}],
		"rollback":{"summary":"set gpu-workers replicas back to 3"},
		"validation":{"expected":["workerReadyReplicas=6"]}
	}`
}
