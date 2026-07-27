package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/approval"
	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/credentials"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/policy"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/sandbox"
	"github.com/tommyxie2026-tech/aicloud/internal/toolgateway"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

func secureToolTestHandler() http.Handler {
	registry := toolgateway.NewMemoryRegistry(
		toolgateway.Definition{
			ID: "repo-inspect", Version: "v1", Image: "alpine/git:2.45.2",
			Command: []string{"git", "status", "--short"}, RiskLevel: "low",
			Permission: "repository:read", CredentialTTL: time.Minute,
			CPU: "250m", Memory: "256Mi", Timeout: time.Minute,
			NetworkMode: sandbox.NetworkDenyAll,
		},
		toolgateway.Definition{
			ID: "workspace-command", Version: "v1", Image: "alpine:3.20",
			Command: []string{"/bin/echo"}, RiskLevel: "high",
			Permission: "workspace:write", CredentialTTL: time.Minute,
			CPU: "250m", Memory: "256Mi", Timeout: time.Minute,
			NetworkMode: sandbox.NetworkDenyAll, WorkspaceWrite: true,
		},
	)
	engine := policy.StaticEngine{Version: "test-policy-v1", Rules: []policy.Rule{
		{Name: "inspect", Subject: "*", Action: "inspect", Resource: "repo-inspect", Allowed: true, Reason: "read only"},
		{Name: "write", Subject: "*", Action: "execute", Resource: "workspace-command", Allowed: true, RequireApproval: true, Reason: "approval required"},
	}}
	auditStore := audit.NewMemoryStore()
	tools := toolgateway.NewService(
		registry,
		engine,
		approval.NewMemoryStore(),
		credentials.NewMemoryBroker(),
		sandbox.NewPlanningExecutor(),
		auditStore,
	)
	control := controlplane.New(
		modelservice.New(repository.NewMemoryModels()),
		repository.NewMemoryTasks(),
		workflow.NoopEngine{},
	).WithGovernance(
		repository.NewMemoryRouteDecisions(),
		repository.NewMemoryCostEvents(),
	).WithSecureTools(tools, auditStore)
	return New(control, logging.New("ERROR")).Handler()
}

func createSecureToolTask(t *testing.T, handler http.Handler) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"input": "inspect repository", "agentId": "agent-1"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("create task status = %d body=%s", response.Code, response.Body.String())
	}
	var task struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	return task.ID
}

func TestSecureToolHTTPFlow(t *testing.T) {
	handler := secureToolTestHandler()

	toolsResponse := httptest.NewRecorder()
	handler.ServeHTTP(toolsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil))
	if toolsResponse.Code != http.StatusOK {
		t.Fatalf("tools status = %d body=%s", toolsResponse.Code, toolsResponse.Body.String())
	}
	var tools []toolgateway.Definition
	if err := json.Unmarshal(toolsResponse.Body.Bytes(), &tools); err != nil {
		t.Fatalf("decode tools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("tool count = %d", len(tools))
	}

	taskID := createSecureToolTask(t, handler)
	inspectBody, _ := json.Marshal(map[string]any{
		"action": "inspect", "workspacePath": "/workspace/repo",
	})
	inspectResponse := httptest.NewRecorder()
	handler.ServeHTTP(inspectResponse, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID+"/tools/repo-inspect",
		bytes.NewReader(inspectBody),
	))
	if inspectResponse.Code != http.StatusCreated {
		t.Fatalf("inspect status = %d body=%s", inspectResponse.Code, inspectResponse.Body.String())
	}
	var result toolgateway.Result
	if err := json.Unmarshal(inspectResponse.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.SandboxResult.Status != "PLANNED" || result.SandboxResult.Plan == nil {
		t.Fatalf("unexpected sandbox result: %#v", result.SandboxResult)
	}

	writeBody, _ := json.Marshal(map[string]any{
		"action": "execute", "arguments": []string{"change"}, "workspacePath": "/workspace/repo",
	})
	writeResponse := httptest.NewRecorder()
	handler.ServeHTTP(writeResponse, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID+"/tools/workspace-command",
		bytes.NewReader(writeBody),
	))
	if writeResponse.Code != http.StatusConflict {
		t.Fatalf("write status = %d body=%s", writeResponse.Code, writeResponse.Body.String())
	}

	auditResponse := httptest.NewRecorder()
	handler.ServeHTTP(auditResponse, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+taskID+"/audit", nil))
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("audit status = %d body=%s", auditResponse.Code, auditResponse.Body.String())
	}
	var events []audit.Event
	if err := json.Unmarshal(auditResponse.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("audit event count = %d: %#v", len(events), events)
	}
	if events[0].Status != "PLANNED" || events[1].Status != "WAITING_APPROVAL" {
		t.Fatalf("unexpected audit statuses: %#v", events)
	}
}

func TestSecureToolRejectsSpoofedTaskContext(t *testing.T) {
	handler := secureToolTestHandler()
	taskID := createSecureToolTask(t, handler)
	body, _ := json.Marshal(map[string]any{
		"action": "inspect", "agentId": "agent-1", "workspacePath": "/workspace/repo",
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks/"+taskID+"/tools/repo-inspect",
		bytes.NewReader(body),
	).WithContext(context.Background()))
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
}
