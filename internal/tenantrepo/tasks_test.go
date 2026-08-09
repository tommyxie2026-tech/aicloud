package tenantrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/tenant"
)

func TestScopedTasksPreventsCrossTenantReads(t *testing.T) {
	base := repository.NewMemoryTasks()
	owners := NewMemoryOwnershipStore()
	tasks := NewScopedTasks(base, owners)

	ctxA := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", ProjectID: "project-a", SubjectID: "user-a"})
	ctxB := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-b", ProjectID: "project-b", SubjectID: "user-b"})

	created, err := tasks.Create(ctxA, domain.Task{ID: "task-1", Input: "test", Status: domain.TaskPending, TraceID: "trace-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
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

func TestScopedTasksAllowsTrustedSystemContext(t *testing.T) {
	base := repository.NewMemoryTasks()
	tasks := NewScopedTasks(base, NewMemoryOwnershipStore())
	_, err := tasks.Create(context.Background(), domain.Task{ID: "system-task", Input: "bootstrap", Status: domain.TaskPending})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := tasks.Get(context.Background(), "system-task"); err != nil {
		t.Fatalf("system Get returned error: %v", err)
	}
}
