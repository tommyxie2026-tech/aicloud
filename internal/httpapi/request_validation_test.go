package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTaskRejectsUnknownJSONField(t *testing.T) {
	handler := taskCommandTestHandler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(`{"input":"review repository","unknown":true}`))
	request.Header.Set(idempotencyKeyHeader, "closed-request-test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope.Error.Code != "INVALID_REQUEST" || envelope.Error.RequestID == "" || envelope.Error.TraceID == "" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}
