package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestMetadataPreservesValidHeaders(t *testing.T) {
	handler := WithRequestMetadata(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := RequestMetadataFromContext(r.Context())
		if metadata.RequestID != "request-123" || metadata.TraceID != "trace-456" {
			t.Fatalf("unexpected metadata: %+v", metadata)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set(RequestIDHeader, "request-123")
	request.Header.Set(TraceIDHeader, "trace-456")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
	if response.Header().Get(RequestIDHeader) != "request-123" || response.Header().Get(TraceIDHeader) != "trace-456" {
		t.Fatal("response correlation headers missing")
	}
}

func TestRequestMetadataGeneratesForInvalidHeaders(t *testing.T) {
	handler := WithRequestMetadata(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := RequestMetadataFromContext(r.Context())
		if metadata.RequestID == "" || metadata.TraceID == "" {
			t.Fatalf("generated metadata must be non-empty: %+v", metadata)
		}
		if metadata.RequestID == "bad value" || metadata.TraceID == "bad/value" {
			t.Fatalf("invalid values were retained: %+v", metadata)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set(RequestIDHeader, "bad value")
	request.Header.Set(TraceIDHeader, "bad/value")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}
