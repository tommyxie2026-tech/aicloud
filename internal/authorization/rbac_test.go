package authorization

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func testPrincipal(role, tenant, project string) identity.Principal {
	return identity.Principal{Type: identity.PrincipalUser, SubjectID: "user-1", TenantID: tenant, ProjectID: project, Roles: []string{role}, AuthnMethod: "test", Issuer: "test"}
}

func TestDeveloperCanCreateTaskInOwnProject(t *testing.T) {
	decision, err := NewDefault().Authorize(context.Background(), Request{Principal: testPrincipal(RoleDeveloper, "tenant-a", "project-a"), Action: ActionTaskCreate, Resource: Resource{Kind: "task", Scope: ScopeProject, TenantID: "tenant-a", ProjectID: "project-a"}})
	if err != nil || !decision.Allowed || decision.MatchedRole != RoleDeveloper {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
}

func TestViewerCannotMutateTask(t *testing.T) {
	decision, err := NewDefault().Authorize(context.Background(), Request{Principal: testPrincipal(RoleViewer, "tenant-a", "project-a"), Action: ActionTaskRoute, Resource: Resource{Kind: "task", Scope: ScopeProject}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Layer != "rbac" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestTenantAdminCannotCrossProjectAttributeBoundary(t *testing.T) {
	decision, err := NewDefault().Authorize(context.Background(), Request{Principal: testPrincipal(RoleTenantAdmin, "tenant-a", "project-a"), Action: ActionTaskRead, Resource: Resource{Kind: "task", Scope: ScopeProject, TenantID: "tenant-a", ProjectID: "project-b"}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed || decision.Layer != "abac" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestNoRoleFailsClosed(t *testing.T) {
	principal := testPrincipal("", "tenant-a", "project-a")
	principal.Roles = nil
	decision, err := NewDefault().Authorize(context.Background(), Request{Principal: principal, Action: ActionTaskRead, Resource: Resource{Kind: "task", Scope: ScopeProject}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatalf("unexpected allow: %+v", decision)
	}
}
