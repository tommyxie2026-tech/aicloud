package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestTrustedIngressRejectsUnscopedAPIRequest(t *testing.T) {
	handler := WithTrustedIngressPrincipal(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unscoped request reached handler")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusUnauthorized)
	}
}

func TestTrustedIngressPropagatesPrincipal(t *testing.T) {
	handler := WithTrustedIngressPrincipal(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := identity.RequireProject(r.Context())
		if err != nil {
			t.Fatalf("RequireProject returned error: %v", err)
		}
		if principal.Type != identity.PrincipalServiceAccount || principal.TenantID != "tenant-a" || principal.ProjectID != "project-a" || principal.SubjectID != "svc-a" {
			t.Fatalf("unexpected principal: %+v", principal)
		}
		if !principal.HasCapability("task:create") {
			t.Fatalf("capabilities not propagated: %+v", principal.Capabilities)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set(TenantHeader, "tenant-a")
	request.Header.Set(ProjectHeader, "project-a")
	request.Header.Set(SubjectHeader, "svc-a")
	request.Header.Set(PrincipalTypeHeader, string(identity.PrincipalServiceAccount))
	request.Header.Set(CapabilitiesHeader, "task:create, task:read")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}

func TestTrustedIngressRejectsSystemPrincipalHeader(t *testing.T) {
	handler := WithTrustedIngressPrincipal(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("external system principal reached handler")
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set(TenantHeader, "tenant-a")
	request.Header.Set(ProjectHeader, "project-a")
	request.Header.Set(SubjectHeader, "system-a")
	request.Header.Set(PrincipalTypeHeader, string(identity.PrincipalSystem))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusUnauthorized)
	}
}

func TestTrustedIngressDoesNotGateHealthEndpoints(t *testing.T) {
	handler := WithTrustedIngressPrincipal(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}
