package tenantrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

// ScopedTasks enforces the frozen S0 Task identity contract at the repository
// boundary. Tenant, Project and CreatedBy are persisted on Task itself; the old
// task_ownership side table is no longer a runtime source of truth.
type ScopedTasks struct {
	base domain.TaskRepository
}

func NewScopedTasks(base domain.TaskRepository) *ScopedTasks {
	return &ScopedTasks{base: base}
}

func (r *ScopedTasks) List(ctx context.Context) ([]domain.Task, error) {
	principal, err := requireProjectPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := r.base.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.Task, 0, len(items))
	for _, task := range items {
		if principal.OwnsProject(task.TenantID, task.ProjectID) {
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

func (r *ScopedTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	principal, err := requireProjectPrincipal(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	task, err := r.base.Get(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	if !principal.OwnsProject(task.TenantID, task.ProjectID) {
		return domain.Task{}, repository.ErrNotFound
	}
	return task, nil
}

func (r *ScopedTasks) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	principal, err := requireProjectPrincipal(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	if task.TenantID != "" && task.TenantID != principal.TenantID {
		return domain.Task{}, fmt.Errorf("task tenant identity does not match principal")
	}
	if task.ProjectID != "" && task.ProjectID != principal.ProjectID {
		return domain.Task{}, fmt.Errorf("task project identity does not match principal")
	}
	if task.CreatedBy != "" && task.CreatedBy != principal.SubjectID {
		return domain.Task{}, fmt.Errorf("task creator identity does not match principal")
	}
	task.TenantID = principal.TenantID
	task.ProjectID = principal.ProjectID
	task.CreatedBy = principal.SubjectID
	return r.base.Create(ctx, task)
}

func (r *ScopedTasks) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	principal, err := requireProjectPrincipal(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	current, err := r.base.Get(ctx, task.ID)
	if err != nil {
		return domain.Task{}, err
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return domain.Task{}, repository.ErrNotFound
	}
	if task.TenantID != current.TenantID || task.ProjectID != current.ProjectID || task.CreatedBy != current.CreatedBy {
		return domain.Task{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	return r.base.Update(ctx, task)
}

func requireProjectPrincipal(ctx context.Context) (identity.Principal, error) {
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return identity.Principal{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return identity.Principal{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	return principal, nil
}

// IsNotFoundOrIdentityError is useful to callers that deliberately collapse
// cross-tenant authorization into not-found semantics while retaining explicit
// authentication errors for missing Principal context.
func IsNotFoundOrIdentityError(err error) bool {
	return errors.Is(err, repository.ErrNotFound) ||
		errors.Is(err, identity.ErrPrincipalRequired) ||
		errors.Is(err, identity.ErrTenantRequired) ||
		errors.Is(err, identity.ErrProjectRequired)
}
