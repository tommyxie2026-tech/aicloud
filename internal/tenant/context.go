package tenant

import (
	"context"
	"errors"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

var ErrScopeRequired = errors.New("tenant scope is required")

// Scope is a compatibility projection of identity.Principal. New production
// code should consume identity.Principal directly.
type Scope struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId,omitempty"`
	SubjectID string `json:"subjectId,omitempty"`
}

// WithScope is retained for compatibility with older tests and internal code.
// It creates an explicit User Principal; it never creates or implies System
// authority.
func WithScope(ctx context.Context, scope Scope) context.Context {
	scope = normalize(scope)
	principal := identity.Principal{
		Type:        identity.PrincipalUser,
		SubjectID:   scope.SubjectID,
		TenantID:    scope.TenantID,
		ProjectID:   scope.ProjectID,
		AuthnMethod: "legacy_scope",
		Issuer:      "aicloud-compat",
	}
	return identity.WithPrincipal(ctx, principal)
}

func FromContext(ctx context.Context) (Scope, bool) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || principal.TenantID == "" {
		return Scope{}, false
	}
	return Scope{
		TenantID:  principal.TenantID,
		ProjectID: principal.ProjectID,
		SubjectID: principal.SubjectID,
	}, true
}

func Require(ctx context.Context) (Scope, error) {
	scope, ok := FromContext(ctx)
	if !ok {
		return Scope{}, ErrScopeRequired
	}
	return scope, nil
}

func OwnsTask(scope Scope, tenantID, projectID string) bool {
	if strings.TrimSpace(scope.TenantID) == "" || scope.TenantID != strings.TrimSpace(tenantID) {
		return false
	}
	if strings.TrimSpace(scope.ProjectID) == "" || scope.ProjectID != strings.TrimSpace(projectID) {
		return false
	}
	return true
}

func normalize(scope Scope) Scope {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.ProjectID = strings.TrimSpace(scope.ProjectID)
	scope.SubjectID = strings.TrimSpace(scope.SubjectID)
	return scope
}
