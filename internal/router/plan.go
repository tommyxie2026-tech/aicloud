package router

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

// Plan evaluates routing policy without persisting the RouteDecision. R6 uses
// this pure planning boundary so the decision can be committed atomically with
// the Task routing transition, TaskEvent, Outbox and command result.
func (r *Router) Plan(ctx context.Context, req Request) (domain.RouteDecision, error) {
	if r == nil {
		return domain.RouteDecision{}, fmt.Errorf("router is required")
	}
	if deployments := r.deploymentRepository(); deployments != nil {
		return r.PlanWithDeployments(ctx, req, deployments)
	}
	planner := *r
	planner.decisions = nil
	return planner.Decide(ctx, req)
}
