package httpapi

import (
	"net/http"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func resolveModelAccess(r *http.Request, principal identity.Principal, parts []string) (authorization.Request, []string, bool) {
	if len(parts) < 3 || parts[2] != "models" {
		return authorization.Request{}, nil, false
	}
	if len(parts) == 3 {
		return accessRequest(r, principal, tenantAccessResource(principal, "model", ""),
			methodAction{http.MethodGet, authorization.ActionModelRead},
			methodAction{http.MethodPost, authorization.ActionModelWrite})
	}
	if len(parts) == 4 && parts[3] != "" {
		return accessRequest(r, principal, tenantAccessResource(principal, "model", parts[3]),
			methodAction{http.MethodGet, authorization.ActionModelRead},
			methodAction{http.MethodPut, authorization.ActionModelWrite})
	}
	if len(parts) == 5 && parts[3] != "" && parts[4] == "admission" {
		return accessRequest(r, principal, tenantAccessResource(principal, "model_admission", parts[3]),
			methodAction{http.MethodGet, authorization.ActionModelAdmissionRead},
			methodAction{http.MethodPost, authorization.ActionModelAdmissionWrite})
	}
	return authorization.Request{}, nil, false
}
