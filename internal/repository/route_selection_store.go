package repository

import (
	"context"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

// RouteSelectionCommit persists one routing result while the canonical Task
// remains in ROUTING. The workflow owns the later ROUTING -> EXECUTING state
// transition; this command owns only the selected route evidence.
type RouteSelectionCommit struct {
	Task        domain.Task
	Decision    domain.RouteDecision
	Policy      execution.PolicyDecisionRecord
	Target      execution.ExecutionTarget
	Event       domain.TaskEvent
	Idempotency domain.IdempotencyRecord
}

type RouteSelectionResult struct {
	Task        domain.Task
	Decision    domain.RouteDecision
	Policy      execution.PolicyDecisionRecord
	Target      execution.ExecutionTarget
	Event       domain.TaskEvent
	Idempotency domain.IdempotencyRecord
	Replayed    bool
}

// RouteSelectionStore closes the ECP routing dual-write boundary without
// replacing the existing PLANNING -> ROUTING command. Implementations must
// atomically persist Task projection metadata, RouteDecision, frozen policy and
// target evidence, TaskEvent and idempotency response. ResolveRouteSelection
// must be called before volatile routing work so commit-before-ack retries replay
// the exact approved target instead of recomputing policy, health, capacity or
// pricing signals.
type RouteSelectionStore interface {
	ResolveRouteSelection(context.Context, IdempotencyLookup) (RouteSelectionResult, bool, error)
	CommitRouteSelection(context.Context, RouteSelectionCommit) (RouteSelectionResult, error)
}
