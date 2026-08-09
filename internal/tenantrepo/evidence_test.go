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

func TestEvidenceStoresInheritTaskOwnership(t *testing.T) {
	baseTasks := repository.NewMemoryTasks()
	tasks := NewScopedTasks(baseTasks, NewMemoryOwnershipStore())
	routes := NewScopedRouteDecisions(repository.NewMemoryRouteDecisions(), tasks)
	costs := NewScopedCostEvents(repository.NewMemoryCostEvents(), tasks)

	ctxA := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", ProjectID: "project-a", SubjectID: "user-a"})
	ctxB := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-b", ProjectID: "project-b", SubjectID: "user-b"})
	now := time.Now().UTC()
	if _, err := tasks.Create(ctxA, domain.Task{ID: "task-1", Input: "test", Status: domain.TaskPending, TraceID: "trace-1", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	if _, err := routes.Create(ctxA, domain.RouteDecision{ID: "route-1", TaskID: "task-1", CreatedAt: now}); err != nil {
		t.Fatalf("Create route: %v", err)
	}
	if _, err := costs.Append(ctxA, domain.CostEvent{ID: "cost-1", TaskID: "task-1", TraceID: "trace-1", Component: domain.CostModelInput, Currency: "USD", CreatedAt: now}); err != nil {
		t.Fatalf("Append cost: %v", err)
	}

	if _, err := routes.ListByTask(ctxB, "task-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-tenant route error=%v want ErrNotFound", err)
	}
	if _, err := costs.ListByTask(ctxB, "task-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-tenant cost error=%v want ErrNotFound", err)
	}
}
