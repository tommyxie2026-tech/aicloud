package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type staticPrincipalVerifier struct {
	principal identity.Principal
	err       error
}

func (v staticPrincipalVerifier) Verify(*http.Request) (identity.Principal, error) {
	return v.principal, v.err
}

func TestPrincipalVerifierInjectsVerifiedPrincipal(t *testing.T) {
	verifier := staticPrincipalVerifier{principal: identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a",
		ProjectID: "project-a", AuthnMethod: "test", Issuer: "test-issuer",
	}}
	handler := WithPrincipalVerifier(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := identity.RequireProject(r.Context())
		if err != nil {
			t.Fatalf("RequireProject returned error: %v", err)
		}
		if principal.SubjectID != "user-a" || principal.ProjectID != "project-a" {
			t.Fatalf("unexpected principal: %+v", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusNoContent)
	}
}

func TestMissingVerifierFailsClosedWithErrorEnvelope(t *testing.T) {
	handler := WithRequestMetadata(WithPrincipalVerifier(nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached protected handler")
	})))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusInternalServerError)
	}
	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Error.Code != "AUTHENTICATION_NOT_CONFIGURED" || envelope.Error.RequestID == "" || envelope.Error.TraceID == "" || envelope.Error.Retryable {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestTrustedIngressAuthenticationFailureUsesStableEnvelope(t *testing.T) {
	handler := WithRequestMetadata(WithTrustedIngressPrincipal(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid request reached handler")
	})))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", response.Code, http.StatusUnauthorized)
	}
	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Error.Code != "UNAUTHENTICATED" || envelope.Error.RequestID == "" || envelope.Error.TraceID == "" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}
