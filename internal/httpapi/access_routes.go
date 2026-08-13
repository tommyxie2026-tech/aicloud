package httpapi

import (
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type methodAction struct {
	method string
	action authorization.Action
}

func resolveAPIAccess(r *http.Request, principal identity.Principal) (authorization.Request, []string, bool) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "v1" {
		return authorization.Request{}, nil, false
	}
	if request, methods, ok := resolveModelAccess(r, principal, parts); ok {
		return request, methods, true
	}
	if request, methods, ok := resolveTaskAccess(r, principal, parts); ok {
		return request, methods, true
	}
	if len(parts) == 3 && parts[2] == "tools" {
		return accessRequest(r, principal, tenantAccessResource(principal, "tool", ""), methodAction{http.MethodGet, authorization.ActionToolRead})
	}
	return authorization.Request{}, nil, false
}

func accessRequest(r *http.Request, principal identity.Principal, resource authorization.Resource, allowed ...methodAction) (authorization.Request, []string, bool) {
	methods := make([]string, 0, len(allowed))
	for _, item := range allowed {
		methods = append(methods, item.method)
		if r.Method == item.method {
			return authorization.Request{Principal: principal, Action: item.action, Resource: resource}, methods, true
		}
	}
	return authorization.Request{Principal: principal, Resource: resource}, methods, true
}

func tenantAccessResource(principal identity.Principal, kind, id string) authorization.Resource {
	return authorization.Resource{Kind: kind, ID: id, Scope: authorization.ScopeTenant, TenantID: principal.TenantID}
}

func projectAccessResource(principal identity.Principal, kind, id string) authorization.Resource {
	return authorization.Resource{Kind: kind, ID: id, Scope: authorization.ScopeProject, TenantID: principal.TenantID, ProjectID: principal.ProjectID}
}
