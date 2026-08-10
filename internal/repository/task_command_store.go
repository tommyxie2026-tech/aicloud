package repository

import "context"

// TaskCommandStore is the R6 transaction-level persistence contract used by
// the Control Plane. Implementations must preserve the atomicity guarantees of
// Task projection + TaskEvent + Outbox + Idempotency described by the frozen
// contracts.
type TaskCommandStore interface {
	CreateTask(context.Context, TaskCreateCommit) (TaskCommandCommitResult, error)
	CommitTransition(context.Context, TaskCommandCommit) (TaskCommandCommitResult, error)
}

// TaskCommandStoreProvider lets a scoped Task repository expose an optional
// durable command kernel without changing the legacy TaskRepository contract.
// Development/in-memory repositories may return nil; production PostgreSQL
// repositories expose the R6 transactional implementation.
type TaskCommandStoreProvider interface {
	TaskCommands() TaskCommandStore
}
