package identity

import (
	"context"
	"errors"
	"testing"
)

func TestMissingPrincipalFailsClosed(t *testing.T) {
	if _, err := RequirePrincipal(context.Background()); !errors.Is(err, ErrPrincipalRequired) {
		t.Fatalf("RequirePrincipal error=%v want ErrPrincipalRequired", err)
	}
}

func TestUserRequiresTenantAndProjectExplicitly(t *testing.T) {
	ctx := WithPrincipal(context.Background(), Principal{
		Type: PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a",
		AuthnMethod: "trusted_ingress", Issuer: "test",
	})
	if _, err := RequireTenant(ctx); err != nil {
		t.Fatalf("RequireTenant returned error: %v", err)
	}
	if _, err := RequireProject(ctx); !errors.Is(err, ErrProjectRequired) {
		t.Fatalf("RequireProject error=%v want ErrProjectRequired", err)
	}
}

func TestSystemPrincipalMustBeExplicitAndCapable(t *testing.T) {
	principal, err := NewSystemPrincipal("reconciler", "repair task evidence", CapabilityTaskSystemAccess)
	if err != nil {
		t.Fatalf("NewSystemPrincipal returned error: %v", err)
	}
	ctx := WithPrincipal(context.Background(), principal)
	if _, err := RequireSystemCapability(ctx, CapabilityTaskSystemAccess); err != nil {
		t.Fatalf("RequireSystemCapability returned error: %v", err)
	}
	if _, err := RequireSystemCapability(ctx, CapabilityDatabaseAdmin); !errors.Is(err, ErrCapabilityRequired) {
		t.Fatalf("missing capability error=%v want ErrCapabilityRequired", err)
	}
}
