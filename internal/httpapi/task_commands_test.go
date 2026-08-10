package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

func taskCommandTestHandler() http.Handler {
	control := controlplane.New(modelservice.New(repository.NewMemoryModels()), repository.NewMemoryTasks(), workflow.NoopEngine{})
	return New(control, logging.New("ERROR")).FullHandler()
}

func TestTaskCreationRequiresIdempotencyKey(t *testing.T) {
	handler := taskCommandTestHandler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"input":"review repository"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestTaskCreationAcceptsIdempotencyKey(t *testing.T) {
	handler := taskCommandTestHandler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"input":"review repository","agentId":"agent-a"}`))
	request.Header.Set(idempotencyKeyHeader, "create-task-a")
	request.Header.Set("X-Request-ID", "request-a")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d want=%d body=%s", response.Code, http.StatusAccepted, response.Body.String())
	}
}

func TestCanonicalRequestDigestStableForSameBusinessRequest(t *testing.T) {
	type request struct {
		Input   string `json:"input"`
		AgentID string `json:"agentId"`
	}
	first, err := canonicalRequestDigest(request{Input: "review repository", AgentID: "agent-a"})
	if err != nil {
		t.Fatalf("first digest: %v", err)
	}
	second, err := canonicalRequestDigest(request{Input: "review repository", AgentID: "agent-a"})
	if err != nil {
		t.Fatalf("second digest: %v", err)
	}
	if first != second || first == "" {
		t.Fatalf("digest first=%q second=%q", first, second)
	}
}
