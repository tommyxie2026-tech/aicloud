package router

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

// Plan evaluates the same routing policy as Decide but deliberately does not
// persist the RouteDecision. R6 uses this pure planning boundary so the chosen
// decision can be committed atomically with the Task routing transition,
// TaskEvent, Outbox and command Idempotency result.
func (r *Router) Plan(ctx context.Context, req Request) (domain.RouteDecision, error) {
	if r == nil {
		return domain.RouteDecision{}, fmt.Errorf("router is required")
	}
	planner := *r
	planner.decisions = nil
	return planner.Decide(ctx, req)
}
