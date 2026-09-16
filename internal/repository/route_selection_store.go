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
	Target      execution.ExecutionTarget
	Event       domain.TaskEvent
	Idempotency domain.IdempotencyRecord
}

type RouteSelectionResult struct {
	Task        domain.Task
	Decision    domain.RouteDecision
	Target      execution.ExecutionTarget
	Event       domain.TaskEvent
	Idempotency domain.IdempotencyRecord
	Replayed    bool
}

// RouteSelectionStore closes the ECP routing dual-write boundary without
// replacing the existing PLANNING -> ROUTING command. Implementations must
// atomically persist Task projection metadata, RouteDecision, TaskEvent and
// idempotency response.
type RouteSelectionStore interface {
	CommitRouteSelection(context.Context, RouteSelectionCommit) (RouteSelectionResult, error)
}
