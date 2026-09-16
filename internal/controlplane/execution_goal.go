package controlplane

import (
	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/executionbinding"
)

// GoalSnapshotOptions remains as a compatibility alias for callers that already
// use the controlplane helper. The canonical conversion lives in the neutral
// executionbinding package so workflow code can reuse it without an import cycle.
type GoalSnapshotOptions = executionbinding.GoalSnapshotOptions

func GoalSnapshotFromTask(task domain.Task, options GoalSnapshotOptions) (execution.Goal, error) {
	return executionbinding.GoalSnapshotFromTask(task, options)
}
