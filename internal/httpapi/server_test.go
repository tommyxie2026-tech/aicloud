package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandler() http.Handler {
	control := controlplane.New(modelservice.New(repository.NewMemoryModels()), repository.NewMemoryTasks(), workflow.NoopEngine{})
	return New(control, logging.New("ERROR")).Handler()
}
func TestHealthz(t *testing.T) {
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
}
func TestModelsAndTasks(t *testing.T) {
	handler := testHandler()
	modelBody, _ := json.Marshal(map[string]any{"id": "m1", "name": "Model One", "provider": "mock"})
	modelResponse := httptest.NewRecorder()
	handler.ServeHTTP(modelResponse, httptest.NewRequest(http.MethodPost, "/api/v1/models", bytes.NewReader(modelBody)))
	if modelResponse.Code != http.StatusCreated {
		t.Fatalf("model status = %d, body = %s", modelResponse.Code, modelResponse.Body.String())
	}
	taskBody, _ := json.Marshal(map[string]string{"input": "review repository"})
	taskResponse := httptest.NewRecorder()
	handler.ServeHTTP(taskResponse, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(taskBody)))
	if taskResponse.Code != http.StatusAccepted {
		t.Fatalf("task status = %d", taskResponse.Code)
	}
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d", listResponse.Code)
	}
}
