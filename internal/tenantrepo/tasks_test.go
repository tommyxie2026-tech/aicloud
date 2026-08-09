package tenantrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func projectContext(tenantID, projectID, subjectID string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalUser, SubjectID: subjectID,
		TenantID: tenantID, ProjectID: projectID,
		AuthnMethod: "test", Issuer: "test",
	})
}

func TestScopedTasksPreventsCrossTenantReads(t *testing.T) {
	base := repository.NewMemoryTasks()
	tasks := NewScopedTasks(base)
	ctxA := projectContext("tenant-a", "project-a", "user-a")
	ctxB := projectContext("tenant-b", "project-b", "user-b")

	created, err := tasks.Create(ctxA, domain.Task{ID: "task-1", Input: "test", Status: domain.TaskPending, TraceID: "trace-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.TenantID != "tenant-a" || created.ProjectID != "project-a" || created.CreatedBy != "user-a" {
		t.Fatalf("task identity not bound atomically: %+v", created)
	}
	if _, err := tasks.Get(ctxA, created.ID); err != nil {
		t.Fatalf("owner Get returned error: %v", err)
	}
	if _, err := tasks.Get(ctxB, created.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-tenant Get error=%v want ErrNotFound", err)
	}
	items, err := tasks.List(ctxB)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("cross-tenant List returned %d tasks", len(items))
	}
}

func TestScopedTasksMissingPrincipalFailsClosed(t *testing.T) {
	tasks := NewScopedTasks(repository.NewMemoryTasks())
	if _, err := tasks.Create(context.Background(), domain.Task{ID: "task-1"}); !errors.Is(err, identity.ErrPrincipalRequired) {
		t.Fatalf("Create error=%v want ErrPrincipalRequired", err)
	}
	if _, err := tasks.List(context.Background()); !errors.Is(err, identity.ErrPrincipalRequired) {
		t.Fatalf("List error=%v want ErrPrincipalRequired", err)
	}
}

func TestScopedTasksSystemAccessMustBeExplicit(t *testing.T) {
	base := repository.NewMemoryTasks()
	tasks := NewScopedTasks(base)
	ctx := projectContext("tenant-a", "project-a", "user-a")
	created, err := tasks.Create(ctx, domain.Task{ID: "task-1", Status: domain.TaskPending})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	principal, err := identity.NewSystemPrincipal("reconciler", "inspect task evidence", identity.CapabilityTaskSystemAccess)
	if err != nil {
		t.Fatalf("NewSystemPrincipal returned error: %v", err)
	}
	systemCtx := identity.WithPrincipal(context.Background(), principal)
	if _, err := tasks.Get(systemCtx, created.ID); err != nil {
		t.Fatalf("explicit system Get returned error: %v", err)
	}
}

func TestScopedTasksRejectsIdentityMutation(t *testing.T) {
	tasks := NewScopedTasks(repository.NewMemoryTasks())
	ctx := projectContext("tenant-a", "project-a", "user-a")
	created, err := tasks.Create(ctx, domain.Task{ID: "task-1", Status: domain.TaskPending})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	created.ProjectID = "project-b"
	if _, err := tasks.Update(ctx, created); err == nil {
		t.Fatal("expected immutable identity update to fail")
	}
}
