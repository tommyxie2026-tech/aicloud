package taskexec

import "context"

// ExecutionAdapter isolates the control plane from concrete model providers,
// agent frameworks, tool runtimes and human execution backends.
type ExecutionAdapter interface {
	Prepare(ctx context.Context, step PlanStep, run ExecutionRun) error
	Execute(ctx context.Context, step PlanStep, run ExecutionRun) (StepResult, error)
	Suspend(ctx context.Context, runID string) error
	Resume(ctx context.Context, runID string) error
	Cancel(ctx context.Context, runID string) error
	CollectUsage(ctx context.Context, runID string) (Usage, error)
}

type StepResult struct {
	ArtifactRef string
	Usage Usage
	Metadata map[string]string
}
