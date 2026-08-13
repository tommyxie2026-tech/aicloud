package authorization

import (
	"context"
	"testing"
)

func TestTenantScopeMismatchIsDenied(t *testing.T) {
	decision, err := NewDefault().Authorize(context.Background(), Request{
		Principal: testPrincipal(RoleTenantAdmin, "tenant-a", "project-a"),
		Action:    ActionModelRead,
		Resource:  Resource{Kind: "model", Scope: ScopeTenant, TenantID: "tenant-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Layer != "abac" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestProjectScopeMismatchIsDenied(t *testing.T) {
	decision, err := NewDefault().Authorize(context.Background(), Request{
		Principal: testPrincipal(RoleTenantAdmin, "tenant-a", "project-a"),
		Action:    ActionTaskRead,
		Resource:  Resource{Kind: "task", Scope: ScopeProject, TenantID: "tenant-a", ProjectID: "project-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Layer != "abac" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}
