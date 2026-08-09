package tenant

import (
	"context"
	"errors"
	"strings"
)

var ErrScopeRequired = errors.New("tenant scope is required")

type Scope struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId,omitempty"`
	SubjectID string `json:"subjectId,omitempty"`
}

type scopeKey struct{}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, normalize(scope))
}

func FromContext(ctx context.Context) (Scope, bool) {
	if ctx == nil {
		return Scope{}, false
	}
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	if !ok || strings.TrimSpace(scope.TenantID) == "" {
		return Scope{}, false
	}
	return normalize(scope), true
}

func Require(ctx context.Context) (Scope, error) {
	scope, ok := FromContext(ctx)
	if !ok {
		return Scope{}, ErrScopeRequired
	}
	return scope, nil
}

func OwnsTask(scope Scope, tenantID, projectID string) bool {
	if strings.TrimSpace(scope.TenantID) == "" || scope.TenantID != tenantID {
		return false
	}
	if scope.ProjectID != "" && projectID != "" && scope.ProjectID != projectID {
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
