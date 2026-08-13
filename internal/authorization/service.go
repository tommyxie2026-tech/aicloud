package authorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type Action string

const (
	ActionAny                 Action = "*"
	ActionModelRead           Action = "model:read"
	ActionModelWrite          Action = "model:write"
	ActionModelAdmissionRead  Action = "model:admission:read"
	ActionModelAdmissionWrite Action = "model:admission:write"
	ActionToolRead            Action = "tool:read"
	ActionToolExecute         Action = "tool:execute"
	ActionTaskRead            Action = "task:read"
	ActionTaskCreate          Action = "task:create"
	ActionTaskRoute           Action = "task:route"
	ActionTaskModelExecute    Action = "task:model:execute"
	ActionTaskRouteRead       Action = "task:route:read"
	ActionTaskCostRead        Action = "task:cost:read"
	ActionTaskAuditRead       Action = "task:audit:read"
	ActionTaskTraceRead       Action = "task:trace:read"
	ActionTaskEvaluationRead  Action = "task:evaluation:read"
	ActionTaskEvaluationWrite Action = "task:evaluation:write"
)

type Scope string

const (
	ScopeTenant  Scope = "tenant"
	ScopeProject Scope = "project"
)

type Resource struct {
	Kind       string            `json:"kind"`
	ID         string            `json:"id,omitempty"`
	Scope      Scope             `json:"scope"`
	TenantID   string            `json:"tenantId,omitempty"`
	ProjectID  string            `json:"projectId,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Request struct {
	Principal identity.Principal `json:"principal"`
	Action    Action             `json:"action"`
	Resource  Resource           `json:"resource"`
}

type Decision struct {
	Allowed       bool   `json:"allowed"`
	Reason        string `json:"reason"`
	Layer         string `json:"layer"`
	PolicyVersion string `json:"policyVersion,omitempty"`
	MatchedRole   string `json:"matchedRole,omitempty"`
}

type Authorizer interface {
	Authorize(context.Context, Request) (Decision, error)
}

type AttributePolicy interface {
	Evaluate(context.Context, Request) (Decision, error)
}

type Service struct {
	rbac   *RBAC
	policy AttributePolicy
}

func New(rbac *RBAC, policy AttributePolicy) *Service {
	return &Service{rbac: rbac, policy: policy}
}

func NewDefault() *Service {
	return New(BuiltinRBAC(), ScopePolicy{Version: "api-scope-v1"})
}

func (s *Service) Authorize(ctx context.Context, request Request) (Decision, error) {
	if s == nil || s.rbac == nil || s.policy == nil {
		return Decision{}, fmt.Errorf("authorization service is not fully configured")
	}
	if err := identity.Validate(request.Principal); err != nil {
		return Decision{}, fmt.Errorf("authorization principal is invalid: %w", err)
	}
	if strings.TrimSpace(string(request.Action)) == "" {
		return Decision{}, fmt.Errorf("authorization action is required")
	}
	if strings.TrimSpace(request.Resource.Kind) == "" {
		return Decision{}, fmt.Errorf("authorization resource kind is required")
	}

	rbacDecision := s.rbac.Evaluate(request)
	if !rbacDecision.Allowed {
		return rbacDecision, nil
	}
	policyDecision, err := s.policy.Evaluate(ctx, request)
	if err != nil {
		return Decision{}, err
	}
	if !policyDecision.Allowed {
		policyDecision.MatchedRole = rbacDecision.MatchedRole
		return policyDecision, nil
	}
	return Decision{
		Allowed:       true,
		Reason:        "RBAC and attribute policy allowed request",
		Layer:         "composite",
		PolicyVersion: joinVersions(rbacDecision.PolicyVersion, policyDecision.PolicyVersion),
		MatchedRole:   rbacDecision.MatchedRole,
	}, nil
}

func joinVersions(values ...string) string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			items = append(items, value)
		}
	}
	return strings.Join(items, "+")
}
