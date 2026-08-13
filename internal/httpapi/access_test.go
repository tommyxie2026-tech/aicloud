package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func accessPrincipal(role string) identity.Principal {
	return identity.Principal{Type: identity.PrincipalUser, SubjectID: "user-1", TenantID: "tenant-a", ProjectID: "project-a", Roles: []string{role}, AuthnMethod: "test", Issuer: "test"}
}

func serveAuthorized(t *testing.T, role, method, path string) (int, bool) {
	t.Helper()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := WithAuthorization(authorization.NewDefault(), next)
	req := httptest.NewRequest(method, path, nil)
	if role != "" {
		req = req.WithContext(identity.WithPrincipal(req.Context(), accessPrincipal(role)))
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response.Code, called
}

func TestAuthorizationAllowsDeveloperTaskMutation(t *testing.T) {
	status, called := serveAuthorized(t, authorization.RoleDeveloper, http.MethodPost, "/api/v1/tasks")
	if status != http.StatusNoContent || !called {
		t.Fatalf("status=%d called=%v", status, called)
	}
}

func TestAuthorizationDeniesViewerTaskMutation(t *testing.T) {
	status, called := serveAuthorized(t, authorization.RoleViewer, http.MethodPost, "/api/v1/tasks")
	if status != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%v", status, called)
	}
}

func TestAuthorizationRequiresPrincipal(t *testing.T) {
	status, called := serveAuthorized(t, "", http.MethodGet, "/api/v1/tasks")
	if status != http.StatusUnauthorized || called {
		t.Fatalf("status=%d called=%v", status, called)
	}
}

func TestAuthorizationFailsClosedForUnknownPathAndMethod(t *testing.T) {
	status, called := serveAuthorized(t, authorization.RoleTenantAdmin, http.MethodGet, "/api/v1/unknown")
	if status != http.StatusNotFound || called {
		t.Fatalf("unknown path status=%d called=%v", status, called)
	}
	status, called = serveAuthorized(t, authorization.RoleTenantAdmin, http.MethodDelete, "/api/v1/tasks")
	if status != http.StatusMethodNotAllowed || called {
		t.Fatalf("method status=%d called=%v", status, called)
	}
}

func TestCurrentPublicRouteActionsResolve(t *testing.T) {
	principal := accessPrincipal(authorization.RoleTenantAdmin)
	cases := []struct {
		method string
		path   string
		action authorization.Action
	}{
		{http.MethodGet, "/api/v1/models", authorization.ActionModelRead},
		{http.MethodPost, "/api/v1/models", authorization.ActionModelWrite},
		{http.MethodGet, "/api/v1/models/m1", authorization.ActionModelRead},
		{http.MethodPut, "/api/v1/models/m1", authorization.ActionModelWrite},
		{http.MethodGet, "/api/v1/models/m1/admission", authorization.ActionModelAdmissionRead},
		{http.MethodPost, "/api/v1/models/m1/admission", authorization.ActionModelAdmissionWrite},
		{http.MethodGet, "/api/v1/tools", authorization.ActionToolRead},
		{http.MethodGet, "/api/v1/tasks", authorization.ActionTaskRead},
		{http.MethodPost, "/api/v1/tasks", authorization.ActionTaskCreate},
		{http.MethodGet, "/api/v1/tasks/t1", authorization.ActionTaskRead},
		{http.MethodPost, "/api/v1/tasks/t1/route", authorization.ActionTaskRoute},
		{http.MethodGet, "/api/v1/tasks/t1/routes", authorization.ActionTaskRouteRead},
		{http.MethodGet, "/api/v1/tasks/t1/costs", authorization.ActionTaskCostRead},
		{http.MethodGet, "/api/v1/tasks/t1/audit", authorization.ActionTaskAuditRead},
		{http.MethodPost, "/api/v1/tasks/t1/model", authorization.ActionTaskModelExecute},
		{http.MethodGet, "/api/v1/tasks/t1/trace", authorization.ActionTaskTraceRead},
		{http.MethodGet, "/api/v1/tasks/t1/evaluations", authorization.ActionTaskEvaluationRead},
		{http.MethodPost, "/api/v1/tasks/t1/evaluations", authorization.ActionTaskEvaluationWrite},
		{http.MethodPost, "/api/v1/tasks/t1/tools/tool-1", authorization.ActionToolExecute},
	}
	for _, item := range cases {
		req := httptest.NewRequest(item.method, item.path, nil)
		resolved, _, known := resolveAPIAccess(req, principal)
		if !known || resolved.Action != item.action {
			t.Fatalf("%s %s resolved=%q known=%v want=%q", item.method, item.path, resolved.Action, known, item.action)
		}
	}
}
