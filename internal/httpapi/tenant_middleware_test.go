package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/tenant"
)

func TestTenantScopeRejectsUnscopedAPIRequest(t *testing.T) {
	handler := WithTenantScope(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unscoped request reached handler")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusUnauthorized)
	}
}

func TestTenantScopePropagatesAuthenticatedScope(t *testing.T) {
	handler := WithTenantScope(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope, err := tenant.Require(r.Context())
		if err != nil {
			t.Fatalf("Require returned error: %v", err)
		}
		if scope.TenantID != "tenant-a" || scope.ProjectID != "project-a" || scope.SubjectID != "user-a" {
			t.Fatalf("unexpected scope: %+v", scope)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set(TenantHeader, "tenant-a")
	request.Header.Set(ProjectHeader, "project-a")
	request.Header.Set(SubjectHeader, "user-a")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}

func TestTenantScopeDoesNotGateHealthEndpoints(t *testing.T) {
	handler := WithTenantScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}
