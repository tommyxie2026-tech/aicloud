package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/tenant"
)

const (
	TenantHeader  = "X-AICloud-Tenant-ID"
	ProjectHeader = "X-AICloud-Project-ID"
	SubjectHeader = "X-AICloud-Subject-ID"
)

// WithTenantScope is the v0.1 trusted-ingress identity boundary.
//
// The ingress/auth proxy is responsible for authenticating the caller and
// replacing these headers. Direct client supplied identity headers must not be
// trusted in production. A later OIDC/JWT verifier can replace the extraction
// mechanism without changing the tenant.Scope contract used by the domain and
// repositories.
func WithTenantScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		scope := tenant.Scope{
			TenantID:  strings.TrimSpace(r.Header.Get(TenantHeader)),
			ProjectID: strings.TrimSpace(r.Header.Get(ProjectHeader)),
			SubjectID: strings.TrimSpace(r.Header.Get(SubjectHeader)),
		}
		if scope.TenantID == "" || scope.SubjectID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "authenticated tenant and subject context are required",
			})
			return
		}

		next.ServeHTTP(w, r.WithContext(tenant.WithScope(r.Context(), scope)))
	})
}
