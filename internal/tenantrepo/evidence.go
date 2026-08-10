package tenantrepo

import (
	"context"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type ScopedRouteDecisions struct {
	base  domain.RouteDecisionRepository
	tasks domain.TaskRepository
}

func NewScopedRouteDecisions(base domain.RouteDecisionRepository, tasks domain.TaskRepository) *ScopedRouteDecisions {
	return &ScopedRouteDecisions{base: base, tasks: tasks}
}

func (r *ScopedRouteDecisions) Create(ctx context.Context, decision domain.RouteDecision) (domain.RouteDecision, error) {
	if _, err := r.tasks.Get(ctx, decision.TaskID); err != nil {
		return domain.RouteDecision{}, err
	}
	return r.base.Create(ctx, decision)
}

func (r *ScopedRouteDecisions) Get(ctx context.Context, id string) (domain.RouteDecision, error) {
	decision, err := r.base.Get(ctx, id)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	if _, err := r.tasks.Get(ctx, decision.TaskID); err != nil {
		return domain.RouteDecision{}, err
	}
	return decision, nil
}

func (r *ScopedRouteDecisions) ListByTask(ctx context.Context, taskID string) ([]domain.RouteDecision, error) {
	if _, err := r.tasks.Get(ctx, taskID); err != nil {
		return nil, err
	}
	return r.base.ListByTask(ctx, taskID)
}

type ScopedCostEvents struct {
	base  domain.CostEventRepository
	tasks domain.TaskRepository
}

func NewScopedCostEvents(base domain.CostEventRepository, tasks domain.TaskRepository) *ScopedCostEvents {
	return &ScopedCostEvents{base: base, tasks: tasks}
}

func (r *ScopedCostEvents) Append(ctx context.Context, event domain.CostEvent) (domain.CostEvent, error) {
	if _, err := r.tasks.Get(ctx, event.TaskID); err != nil {
		return domain.CostEvent{}, err
	}
	return r.base.Append(ctx, event)
}

func (r *ScopedCostEvents) ListByTask(ctx context.Context, taskID string) ([]domain.CostEvent, error) {
	if _, err := r.tasks.Get(ctx, taskID); err != nil {
		return nil, err
	}
	return r.base.ListByTask(ctx, taskID)
}
