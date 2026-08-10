package identity

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type PrincipalType string

const (
	PrincipalUser           PrincipalType = "user"
	PrincipalServiceAccount PrincipalType = "service_account"
	PrincipalSystem         PrincipalType = "system"
)

const (
	CapabilityTaskSystemAccess = "task:system-access"
	CapabilityDatabaseAdmin    = "database:admin"
)

var (
	ErrPrincipalRequired  = errors.New("authenticated principal is required")
	ErrTenantRequired     = errors.New("tenant scope is required")
	ErrProjectRequired    = errors.New("project scope is required")
	ErrCapabilityRequired = errors.New("required capability is missing")
	ErrInvalidPrincipal   = errors.New("invalid principal")
)

// Principal is the verified runtime identity consumed by domain, repository,
// policy, audit and execution layers. HTTP headers and JWT claims must be
// resolved into this type before business code is invoked.
type Principal struct {
	Type         PrincipalType `json:"type"`
	SubjectID    string        `json:"subjectId"`
	TenantID     string        `json:"tenantId,omitempty"`
	ProjectID    string        `json:"projectId,omitempty"`
	Roles        []string      `json:"roles,omitempty"`
	Capabilities []string      `json:"capabilities,omitempty"`
	AuthnMethod  string        `json:"authnMethod"`
	Issuer       string        `json:"issuer,omitempty"`
	SessionID    string        `json:"sessionId,omitempty"`
	Purpose      string        `json:"purpose,omitempty"`
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, normalize(principal))
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	if ctx == nil {
		return Principal{}, false
	}
	principal, ok := ctx.Value(principalKey{}).(Principal)
	if !ok {
		return Principal{}, false
	}
	principal = normalize(principal)
	if Validate(principal) != nil {
		return Principal{}, false
	}
	return principal, true
}

func RequirePrincipal(ctx context.Context) (Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return Principal{}, ErrPrincipalRequired
	}
	return principal, nil
}

func RequireTenant(ctx context.Context) (Principal, error) {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return Principal{}, err
	}
	if principal.TenantID == "" {
		return Principal{}, ErrTenantRequired
	}
	return principal, nil
}

func RequireProject(ctx context.Context) (Principal, error) {
	principal, err := RequireTenant(ctx)
	if err != nil {
		return Principal{}, err
	}
	if principal.ProjectID == "" {
		return Principal{}, ErrProjectRequired
	}
	return principal, nil
}

func RequireCapability(ctx context.Context, capability string) (Principal, error) {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return Principal{}, err
	}
	if !principal.HasCapability(capability) {
		return Principal{}, fmt.Errorf("%w: %s", ErrCapabilityRequired, capability)
	}
	return principal, nil
}

func RequireSystemCapability(ctx context.Context, capability string) (Principal, error) {
	principal, err := RequireCapability(ctx, capability)
	if err != nil {
		return Principal{}, err
	}
	if principal.Type != PrincipalSystem {
		return Principal{}, fmt.Errorf("%w: system principal required", ErrInvalidPrincipal)
	}
	if principal.Purpose == "" {
		return Principal{}, fmt.Errorf("%w: system principal purpose is required", ErrInvalidPrincipal)
	}
	return principal, nil
}

func NewSystemPrincipal(subjectID, purpose string, capabilities ...string) (Principal, error) {
	principal := Principal{
		Type:         PrincipalSystem,
		SubjectID:    subjectID,
		Capabilities: append([]string(nil), capabilities...),
		AuthnMethod:  "internal_system",
		Issuer:       "aicloud",
		Purpose:      purpose,
	}
	if err := Validate(principal); err != nil {
		return Principal{}, err
	}
	return normalize(principal), nil
}

func (p Principal) HasCapability(capability string) bool {
	return slices.Contains(p.Capabilities, strings.TrimSpace(capability))
}

func (p Principal) OwnsProject(tenantID, projectID string) bool {
	if p.TenantID == "" || p.ProjectID == "" {
		return false
	}
	return p.TenantID == strings.TrimSpace(tenantID) && p.ProjectID == strings.TrimSpace(projectID)
}

func Validate(principal Principal) error {
	principal = normalize(principal)
	if principal.SubjectID == "" || principal.AuthnMethod == "" {
		return ErrInvalidPrincipal
	}
	switch principal.Type {
	case PrincipalUser, PrincipalServiceAccount:
		if principal.TenantID == "" {
			return ErrInvalidPrincipal
		}
	case PrincipalSystem:
		if principal.Purpose == "" {
			return ErrInvalidPrincipal
		}
	default:
		return ErrInvalidPrincipal
	}
	return nil
}

func normalize(principal Principal) Principal {
	principal.SubjectID = strings.TrimSpace(principal.SubjectID)
	principal.TenantID = strings.TrimSpace(principal.TenantID)
	principal.ProjectID = strings.TrimSpace(principal.ProjectID)
	principal.AuthnMethod = strings.TrimSpace(principal.AuthnMethod)
	principal.Issuer = strings.TrimSpace(principal.Issuer)
	principal.SessionID = strings.TrimSpace(principal.SessionID)
	principal.Purpose = strings.TrimSpace(principal.Purpose)
	principal.Roles = normalizeStrings(principal.Roles)
	principal.Capabilities = normalizeStrings(principal.Capabilities)
	return principal
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
