package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

func governedTestHandler() http.Handler {
	now := time.Now().UTC()
	models := repository.NewMemoryModels(domain.Model{
		ID:                "model-1",
		Name:              "Model One",
		Version:           "v1",
		Provider:          "test",
		Lifecycle:         domain.ModelActive,
		Capabilities:      []string{"coding"},
		Pricing:           domain.PricingProfile{Currency: "USD", InputPerMillion: 1, OutputPerMillion: 2},
		Health:            domain.HealthHealthy,
		HealthCheckedAt:   &now,
		QuotaRemaining:    100,
		CapacityAvailable: 10,
		ApprovalStatus:    domain.ApprovalApproved,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	tasks := repository.NewMemoryTasks()
	routes := repository.NewMemoryRouteDecisions()
	costs := repository.NewMemoryCostEvents()
	control := controlplane.New(modelservice.New(models), tasks, workflow.NoopEngine{}).WithGovernance(routes, costs)
	return New(control, logging.New("ERROR")).Handler()
}

func TestTaskRoutingAPI(t *testing.T) {
	handler := governedTestHandler()
	taskBody, _ := json.Marshal(map[string]string{"input": "review code"})
	taskResponse := httptest.NewRecorder()
	handler.ServeHTTP(taskResponse, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	if taskResponse.Code != http.StatusAccepted {
		t.Fatalf("task status = %d body=%s", taskResponse.Code, taskResponse.Body.String())
	}
	var task domain.Task
	if err := json.Unmarshal(taskResponse.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}

	routeBody, _ := json.Marshal(map[string]any{
		"routeClass":            "efficient",
		"requiredCapabilities":  []string{"coding"},
		"estimatedInputTokens":  1000,
		"estimatedOutputTokens": 1000,
		"budget":                1,
	})
	routeResponse := httptest.NewRecorder()
	handler.ServeHTTP(routeResponse, httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/route", bytes.NewReader(routeBody)))
	if routeResponse.Code != http.StatusCreated {
		t.Fatalf("route status = %d body=%s", routeResponse.Code, routeResponse.Body.String())
	}
	var decision domain.RouteDecision
	if err := json.Unmarshal(routeResponse.Body.Bytes(), &decision); err != nil {
		t.Fatalf("decode route: %v", err)
	}
	if decision.Selected.ModelID != "model-1" {
		t.Fatalf("selected model = %s", decision.Selected.ModelID)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+task.ID+"/routes", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("route list status = %d", listResponse.Code)
	}
}

func TestModelUpdateAPI(t *testing.T) {
	handler := governedTestHandler()
	body, _ := json.Marshal(domain.Model{
		Name:           "Model One",
		Version:        "v2",
		Provider:       "test",
		Lifecycle:      domain.ModelActive,
		Health:         domain.HealthDegraded,
		ApprovalStatus: domain.ApprovalApproved,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/models/model-1", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", response.Code, response.Body.String())
	}
	var updated domain.Model
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode model: %v", err)
	}
	if updated.Version != "v2" || updated.Health != domain.HealthDegraded {
		t.Fatalf("unexpected model: %#v", updated)
	}
}
