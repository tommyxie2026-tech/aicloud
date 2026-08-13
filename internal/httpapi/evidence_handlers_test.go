package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/admission"
	"github.com/tommyxie2026-tech/aicloud/internal/circuitbreaker"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/evaluation"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelruntime"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func evidenceTestHandler(t *testing.T) http.Handler {
	t.Helper()
	now := time.Now().UTC()
	model := domain.Model{
		ID: "model-1", Name: "Model One", Version: "v1", Provider: "provider-1",
		DeploymentMode: domain.DeploymentPublicAPI, Lifecycle: domain.ModelActive,
		Capabilities: []string{"structured-output"}, Pricing: domain.PricingProfile{Currency: "USD", InputPerMillion: 1, OutputPerMillion: 2},
		Health: domain.HealthHealthy, HealthCheckedAt: &now, QuotaRemaining: -1, CapacityAvailable: -1,
		ServiceTiers: []domain.ServiceTier{domain.TierStandard}, InferenceEfforts: []domain.InferenceEffort{domain.EffortLow},
		ApprovalStatus: domain.ApprovalApproved, CreatedAt: now, UpdatedAt: now,
	}
	models := repository.NewMemoryModels(model)
	tasks := repository.NewMemoryTasks()
	routes := repository.NewMemoryRouteDecisions()
	costEvents := repository.NewMemoryCostEvents()
	traceStore := tracepkg.NewMemoryStore()
	evaluationStore := evaluation.NewMemoryStore()
	admissionService := admission.NewService(admission.NewMemoryStore())
	if err := admissionService.Append(context.Background(), admission.Evidence{
		ID: "evidence-1", ModelID: model.ID, ModelVersion: model.Version, Status: admission.StatusApproved,
		LicenseID: "commercial-api", LicenseTextRef: "https://licenses.example/provider", SourceRef: "https://api.example/v1",
		CommercialUseAllowed: true, HostedServiceAllowed: true, Reviewer: "reviewer", ReviewedAt: &now,
		EvidenceDigest: "sha256:evidence-1", CreatedAt: now,
	}); err != nil {
		t.Fatalf("append admission evidence: %v", err)
	}
	providers := modelruntime.NewMemoryProviderRegistry()
	if err := providers.Put(context.Background(), model.ID, &httpFakeProvider{}); err != nil {
		t.Fatalf("register provider: %v", err)
	}
	runtime := modelruntime.NewExecutor(providers, models, circuitbreaker.New(circuitbreaker.NewMemoryStore(), 2, time.Minute), cost.New(costEvents), traceStore)
	control := controlplane.New(modelservice.New(models), tasks, workflow.NoopEngine{}).
		WithGovernance(routes, costEvents).
		WithEvidence(traceStore, evaluationStore, admissionService, runtime)
	return New(control, logging.New("ERROR")).FullHandler()
}

func TestEvidenceExecutionHTTPWorkflow(t *testing.T) {
	handler := evidenceTestHandler(t)
	taskID := createEvidenceTask(t, handler)

	routeBody, _ := json.Marshal(map[string]any{
		"routeClass": "efficient", "requiredCapabilities": []string{"structured-output"}, "inferenceEffort": "low", "serviceTier": "standard",
		"estimatedInputTokens": 1000, "estimatedOutputTokens": 100, "budget": 1, "evidenceVersion": "eval-config-v1",
		"policyVersion": "route-policy-v1", "requireFreshSignals": true, "signalMaxAgeSeconds": 300,
	})
	routeRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/route", bytes.NewReader(routeBody))
	routeRequest.Header.Set(idempotencyKeyHeader, "evidence-task-route")
	assertHTTPStatus(t, handler, routeRequest, http.StatusCreated)

	modelBody, _ := json.Marshal(map[string]any{"requestId": "request-1", "instruction": "produce JSON", "outputSchema": map[string]string{"name": "result", "version": "v1"}})
	modelRequest := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/model", bytes.NewReader(modelBody))
	modelRequest.Header.Set(idempotencyKeyHeader, "evidence-task-model")
	assertHTTPStatus(t, handler, modelRequest, http.StatusOK)

	evaluationBody, _ := json.Marshal(map[string]any{
		"config": map[string]any{"modelId": "model-1", "modelVersion": "v1", "provider": "provider-1", "promptVersion": "prompt-v1", "workflowVersion": "workflow-v1", "tokenBudget": 10000, "timeBudgetMs": 60000, "retryPolicyVersion": "retry-v1", "sandboxProfile": "sandbox-v1", "datasetId": "developer-agent", "datasetVersion": "dataset-v1", "evaluatorId": "quality", "evaluatorVersion": "evaluator-v1"},
		"metrics": map[string]any{"qualityScore": 0.95, "safetyScore": 0.99, "reliabilityScore": 0.96, "latencyP95Ms": 1000, "costPerSuccessfulTask": 0.01, "humanInterventionRate": 0.01, "taskSuccessRate": 0.98},
		"thresholds": map[string]any{"minimumQuality": 0.90, "minimumSafety": 0.95, "minimumReliability": 0.90, "maximumLatencyP95Ms": 2000, "maximumCostPerSuccess": 0.10, "maximumHumanIntervention": 0.10, "minimumTaskSuccessRate": 0.90},
		"rawOutput": "{\"ok\":true}",
	})
	assertHTTPStatus(t, handler, httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID+"/evaluations", bytes.NewReader(evaluationBody)), http.StatusCreated)

	traceResponse := httptest.NewRecorder()
	handler.ServeHTTP(traceResponse, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/trace", nil))
	if traceResponse.Code != http.StatusOK {
		t.Fatalf("trace status=%d body=%s", traceResponse.Code, traceResponse.Body.String())
	}
	var tracePage struct {
		Items []tracepkg.Event `json:"items"`
	}
	if err := json.Unmarshal(traceResponse.Body.Bytes(), &tracePage); err != nil {
		t.Fatalf("decode trace page: %v", err)
	}
	if len(tracePage.Items) < 4 {
		t.Fatalf("expected task, route, model and evaluation evidence: %#v", tracePage.Items)
	}

	admissionResponse := httptest.NewRecorder()
	handler.ServeHTTP(admissionResponse, httptest.NewRequest(http.MethodGet, "/api/v1/models/model-1/admission", nil))
	if admissionResponse.Code != http.StatusOK {
		t.Fatalf("admission status=%d body=%s", admissionResponse.Code, admissionResponse.Body.String())
	}
	var payload struct {
		Decision admission.Decision `json:"decision"`
	}
	if err := json.Unmarshal(admissionResponse.Body.Bytes(), &payload); err != nil || !payload.Decision.Allowed {
		t.Fatalf("unexpected admission payload: %#v err=%v", payload, err)
	}
}

func assertHTTPStatus(t *testing.T, handler http.Handler, request *http.Request, expected int) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != expected {
		t.Fatalf("status=%d want=%d body=%s", response.Code, expected, response.Body.String())
	}
}

func createEvidenceTask(t *testing.T, handler http.Handler) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"input": "evaluate model", "agentId": "agent-1"})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	request.Header.Set(idempotencyKeyHeader, "evidence-task-create")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("task status=%d body=%s", response.Code, response.Body.String())
	}
	var task domain.Task
	if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	return task.ID
}

type httpFakeProvider struct{}

func (*httpFakeProvider) Name() string                { return "provider-1" }
func (*httpFakeProvider) Type() provider.ProviderType { return provider.ProviderTypeHosted }
func (*httpFakeProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{SupportsStructuredOutput: true}
}
func (*httpFakeProvider) Generate(_ context.Context, request provider.ProviderRequest) (*provider.ProviderResponse, error) {
	return &provider.ProviderResponse{
		RequestID: request.RequestID, ProviderName: "provider-1", ModelName: "model-1", RawText: "{\"ok\":true}",
		Structured: map[string]any{"ok": true}, TokenUsage: provider.TokenUsage{InputTokens: 100, OutputTokens: 20}, FinishReason: "stop",
	}, nil
}
func (*httpFakeProvider) Health(context.Context) (*provider.ProviderHealth, error) {
	return &provider.ProviderHealth{Available: true}, nil
}
