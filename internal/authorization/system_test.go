package authorization

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestPublicAuthorizationDeniesSystemPrincipal(t *testing.T) {
	principal, err := identity.NewSystemPrincipal("worker", "maintenance", identity.CapabilityTaskSystemAccess)
	if err != nil {
		t.Fatal(err)
	}
	principal.Roles = []string{RoleTenantAdmin}
	decision, err := NewDefault().Authorize(context.Background(), Request{
		Principal: principal,
		Action:    ActionTaskRead,
		Resource:  Resource{Kind: "task", Scope: ScopeProject},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatalf("system principal unexpectedly allowed: %+v", decision)
	}
}
