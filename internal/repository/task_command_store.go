package repository

import (
	"context"
	"errors"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var ErrDurableTaskCommandRequired = errors.New("durable task mutation requires the R6 TaskCommandStore")

type IdempotencyLookup struct {
	TenantID      string
	ProjectID     string
	Operation     string
	Key           string
	RequestDigest string
}

// TaskCommandStore is the R6 transaction-level persistence contract used by
// the Control Plane. Implementations must preserve the atomicity guarantees of
// Task projection + TaskEvent + adjacent business records + Outbox +
// Idempotency described by the frozen contracts.
type TaskCommandStore interface {
	CreateTask(context.Context, TaskCreateCommit) (TaskCommandCommitResult, error)
	CommitTransition(context.Context, TaskCommandCommit) (TaskCommandCommitResult, error)
	CommitRouteTransition(context.Context, RouteTaskCommandCommit) (RouteTaskCommandResult, error)
	BeginModelExecution(context.Context, ModelExecutionBeginCommit) (ModelExecutionBeginResult, error)
	FinalizeModelExecution(context.Context, ModelExecutionFinalizeCommit) (ModelExecutionFinalizeResult, error)
	MarkModelExecutionRetryable(context.Context, domain.IdempotencyRecord) error
	ResolveIdempotency(context.Context, IdempotencyLookup) (domain.IdempotencyRecord, bool, error)
}

// TaskCommandStoreProvider lets a scoped Task repository expose an optional
// durable command kernel without changing the legacy TaskRepository contract.
// Development/in-memory repositories may return nil; production PostgreSQL
// repositories expose the R6 transactional implementation.
type TaskCommandStoreProvider interface {
	TaskCommands() TaskCommandStore
}
