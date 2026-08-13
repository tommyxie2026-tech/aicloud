package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type ScopePolicy struct {
	Version string
}

func (p ScopePolicy) Evaluate(_ context.Context, request Request) (Decision, error) {
	version := strings.TrimSpace(p.Version)
	if version == "" {
		version = "api-scope-v1"
	}
	principal := request.Principal
	if principal.Type == identity.PrincipalSystem {
		return Decision{Allowed: false, Reason: "system principals require a dedicated internal authorization path", Layer: "abac", PolicyVersion: version}, nil
	}

	switch request.Resource.Scope {
	case ScopeTenant:
		if principal.TenantID == "" {
			return Decision{Allowed: false, Reason: "tenant scope is required", Layer: "abac", PolicyVersion: version}, nil
		}
		if target := strings.TrimSpace(request.Resource.TenantID); target != "" && target != principal.TenantID {
			return Decision{Allowed: false, Reason: "principal tenant does not match resource tenant", Layer: "abac", PolicyVersion: version}, nil
		}
	case ScopeProject:
		if principal.TenantID == "" || principal.ProjectID == "" {
			return Decision{Allowed: false, Reason: "project scope is required", Layer: "abac", PolicyVersion: version}, nil
		}
		if target := strings.TrimSpace(request.Resource.TenantID); target != "" && target != principal.TenantID {
			return Decision{Allowed: false, Reason: "principal tenant does not match resource tenant", Layer: "abac", PolicyVersion: version}, nil
		}
		if target := strings.TrimSpace(request.Resource.ProjectID); target != "" && target != principal.ProjectID {
			return Decision{Allowed: false, Reason: "principal project does not match resource project", Layer: "abac", PolicyVersion: version}, nil
		}
	default:
		return Decision{}, fmt.Errorf("unsupported authorization resource scope %q", request.Resource.Scope)
	}
	return Decision{Allowed: true, Reason: "principal attributes satisfy resource scope", Layer: "abac", PolicyVersion: version}, nil
}
