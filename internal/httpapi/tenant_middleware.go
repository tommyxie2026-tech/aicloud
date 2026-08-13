package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

const (
	TenantHeader        = "X-AICloud-Tenant-ID"
	ProjectHeader       = "X-AICloud-Project-ID"
	SubjectHeader       = "X-AICloud-Subject-ID"
	PrincipalTypeHeader = "X-AICloud-Principal-Type"
	RolesHeader         = "X-AICloud-Roles"
	CapabilitiesHeader  = "X-AICloud-Capabilities"
	AuthnMethodHeader   = "X-AICloud-Authn-Method"
	IssuerHeader        = "X-AICloud-Issuer"
	SessionHeader       = "X-AICloud-Session-ID"
)

// PrincipalVerifier is the R7 API-boundary authentication contract. Business
// handlers consume only the resulting identity.Principal and remain decoupled
// from trusted headers, JWTs, OIDC discovery and future workload identity.
type PrincipalVerifier interface {
	Verify(*http.Request) (identity.Principal, error)
}

// TrustedIngressVerifier preserves the S1 trusted-header mechanism as an
// explicit compatibility verifier. It is suitable only when an authenticated
// ingress replaces these headers before traffic reaches the API process.
type TrustedIngressVerifier struct{}

func (TrustedIngressVerifier) Verify(r *http.Request) (identity.Principal, error) {
	principalType := identity.PrincipalType(strings.TrimSpace(r.Header.Get(PrincipalTypeHeader)))
	if principalType == "" {
		principalType = identity.PrincipalUser
	}
	if principalType == identity.PrincipalSystem {
		return identity.Principal{}, fmt.Errorf("system principal is not accepted from external ingress")
	}
	if principalType != identity.PrincipalUser && principalType != identity.PrincipalServiceAccount {
		return identity.Principal{}, fmt.Errorf("unsupported principal type")
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
		return identity.Principal{}, fmt.Errorf("authenticated principal context is invalid")
	}
	return principal, nil
}

// WithPrincipalVerifier authenticates public API requests through a pluggable
// verifier and injects the verified Principal into request context. Missing or
// invalid authentication fails closed. Health/readiness endpoints remain
// outside this boundary.
func WithPrincipalVerifier(verifier PrincipalVerifier, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if verifier == nil {
			writeAPIError(w, http.StatusInternalServerError, "AUTHENTICATION_NOT_CONFIGURED", "authentication verifier is not configured", false, nil)
			return
		}

		principal, err := verifier.Verify(r)
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHENTICATED", err.Error(), false, nil)
			return
		}
		if err := identity.Validate(principal); err != nil {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authenticated principal context is invalid", false, nil)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/tasks") && principal.ProjectID == "" {
			writeAPIError(w, http.StatusBadRequest, "PROJECT_CONTEXT_REQUIRED", "project context is required for task APIs", false, nil)
			return
		}

		next.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	})
}

func WithTrustedIngressPrincipal(next http.Handler) http.Handler {
	return WithPrincipalVerifier(TrustedIngressVerifier{}, next)
}

// WithTenantScope is retained as a source-compatible alias during the R7
// migration. New production wiring must use an explicit verifier.
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
