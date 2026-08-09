package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

const (
	TenantHeader       = "X-AICloud-Tenant-ID"
	ProjectHeader      = "X-AICloud-Project-ID"
	SubjectHeader      = "X-AICloud-Subject-ID"
	PrincipalTypeHeader = "X-AICloud-Principal-Type"
	RolesHeader        = "X-AICloud-Roles"
	CapabilitiesHeader = "X-AICloud-Capabilities"
	AuthnMethodHeader  = "X-AICloud-Authn-Method"
	IssuerHeader       = "X-AICloud-Issuer"
	SessionHeader      = "X-AICloud-Session-ID"
)

// WithTrustedIngressPrincipal is the v0.1 authentication compatibility
// boundary. A trusted ingress authenticates the caller and replaces the
// identity headers before traffic reaches this process. Direct client supplied
// identity headers must never be trusted in production.
//
// This middleware always resolves headers into an explicit identity.Principal.
// It never infers System access from missing tenant/project context and never
// accepts an externally supplied System principal.
func WithTrustedIngressPrincipal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		principalType := identity.PrincipalType(strings.TrimSpace(r.Header.Get(PrincipalTypeHeader)))
		if principalType == "" {
			principalType = identity.PrincipalUser
		}
		if principalType == identity.PrincipalSystem {
			writeScopeError(w, http.StatusUnauthorized, "system principal is not accepted from external ingress")
			return
		}
		if principalType != identity.PrincipalUser && principalType != identity.PrincipalServiceAccount {
			writeScopeError(w, http.StatusUnauthorized, "unsupported principal type")
			return
		}

		principal := identity.Principal{
			Type:         principalType,
			SubjectID:    strings.TrimSpace(r.Header.Get(SubjectHeader)),
			TenantID:     strings.TrimSpace(r.Header.Get(TenantHeader)),
			ProjectID:    strings.TrimSpace(r.Header.Get(ProjectHeader)),
			Roles:        splitHeaderList(r.Header.Get(RolesHeader)),
			Capabilities: splitHeaderList(r.Header.Get(CapabilitiesHeader)),
			AuthnMethod:  strings.TrimSpace(r.Header.Get(AuthnMethodHeader)),
			Issuer:       strings.TrimSpace(r.Header.Get(IssuerHeader)),
			SessionID:    strings.TrimSpace(r.Header.Get(SessionHeader)),
		}
		if principal.AuthnMethod == "" {
			principal.AuthnMethod = "trusted_ingress"
		}
		if principal.Issuer == "" {
			principal.Issuer = "trusted-ingress"
		}
		if err := identity.Validate(principal); err != nil {
			writeScopeError(w, http.StatusUnauthorized, "authenticated principal context is invalid")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/tasks") && principal.ProjectID == "" {
			writeScopeError(w, http.StatusBadRequest, "project context is required for task APIs")
			return
		}

		next.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	})
}

// WithTenantScope is retained as a source-compatible alias during the S1
// migration. New wiring must use WithTrustedIngressPrincipal.
func WithTenantScope(next http.Handler) http.Handler {
	return WithTrustedIngressPrincipal(next)
}

func splitHeaderList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			items = append(items, part)
		}
	}
	return items
}

func writeScopeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
