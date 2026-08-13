package httpapi

import (
	"net/http"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func resolveTaskAccess(r *http.Request, principal identity.Principal, parts []string) (authorization.Request, []string, bool) {
	if len(parts) < 3 || parts[2] != "tasks" {
		return authorization.Request{}, nil, false
	}
	if len(parts) == 3 {
		return accessRequest(r, principal, projectAccessResource(principal, "task", ""),
			methodAction{http.MethodGet, authorization.ActionTaskRead},
			methodAction{http.MethodPost, authorization.ActionTaskCreate})
	}
	if len(parts) < 4 || parts[3] == "" {
		return authorization.Request{}, nil, false
	}
	taskID := parts[3]
	if len(parts) == 4 {
		return accessRequest(r, principal, projectAccessResource(principal, "task", taskID),
			methodAction{http.MethodGet, authorization.ActionTaskRead})
	}
	if len(parts) == 5 {
		resource := projectAccessResource(principal, "task", taskID)
		switch parts[4] {
		case "route":
			return accessRequest(r, principal, resource, methodAction{http.MethodPost, authorization.ActionTaskRoute})
		case "routes":
			return accessRequest(r, principal, resource, methodAction{http.MethodGet, authorization.ActionTaskRouteRead})
		case "costs":
			return accessRequest(r, principal, resource, methodAction{http.MethodGet, authorization.ActionTaskCostRead})
		case "audit":
			return accessRequest(r, principal, resource, methodAction{http.MethodGet, authorization.ActionTaskAuditRead})
		case "model":
			return accessRequest(r, principal, resource, methodAction{http.MethodPost, authorization.ActionTaskModelExecute})
		case "trace":
			return accessRequest(r, principal, resource, methodAction{http.MethodGet, authorization.ActionTaskTraceRead})
		case "evaluations":
			return accessRequest(r, principal, resource,
				methodAction{http.MethodGet, authorization.ActionTaskEvaluationRead},
				methodAction{http.MethodPost, authorization.ActionTaskEvaluationWrite})
		}
	}
	if len(parts) == 6 && parts[4] == "tools" && parts[5] != "" {
		resource := projectAccessResource(principal, "tool", parts[5])
		resource.Attributes = map[string]string{"task_id": taskID}
		return accessRequest(r, principal, resource, methodAction{http.MethodPost, authorization.ActionToolExecute})
	}
	return authorization.Request{}, nil, false
}
