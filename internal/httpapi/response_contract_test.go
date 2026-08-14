package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testPage struct {
	Items         []map[string]string `json:"items"`
	NextPageToken string              `json:"nextPageToken"`
}

func TestResponseContractPaginatesCollections(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]string{{"id": "1"}, {"id": "2"}, {"id": "3"}})
	})
	handler := WithRequestMetadata(WithAPIResponseContract(next))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?pageSize=2", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var firstPage testPage
	if err := json.Unmarshal(response.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Items) != 2 || firstPage.NextPageToken == "" {
		t.Fatalf("unexpected page: %+v", firstPage)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/tasks?pageSize=2&pageToken="+firstPage.NextPageToken, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var secondPage testPage
	if err := json.Unmarshal(response.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0]["id"] != "3" || secondPage.NextPageToken != "" {
		t.Fatalf("unexpected second page: %+v", secondPage)
	}
}

func TestResponseContractRejectsInvalidPagination(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := WithRequestMetadata(WithAPIResponseContract(next))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models?pageSize=201", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if called {
		t.Fatal("handler ran for invalid pagination")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != "INVALID_REQUEST" || envelope.Error.RequestID == "" || envelope.Error.TraceID == "" {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
}

func TestResponseContractNormalizesLegacyErrors(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "IDEMPOTENCY_CONFLICT"})
	})
	handler := WithRequestMetadata(WithAPIResponseContract(next))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil))
	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusConflict || envelope.Error.Code != "IDEMPOTENCY_CONFLICT" || envelope.Error.RequestID == "" || envelope.Error.TraceID == "" {
		t.Fatalf("unexpected envelope: status=%d value=%+v", response.Code, envelope)
	}
}

func TestResponseContractAddsTaskResourceVersion(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"id": "task-1", "version": int64(7), "status": "EXECUTING"})
	})
	handler := WithAPIResponseContract(next)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1", nil))
	if response.Header().Get(ResourceVersionHeader) != "7" {
		t.Fatalf("resource version=%q", response.Header().Get(ResourceVersionHeader))
	}
	if response.Header().Get("ETag") != "\"task:task-1:v7\"" {
		t.Fatalf("etag=%q", response.Header().Get("ETag"))
	}
}
