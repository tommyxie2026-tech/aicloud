package repository

import execution "github.com/tommyxie2026-tech/aicloud/execution"

var _ execution.AttemptExecutionCoordinator = (*PostgresExecutionNodeLeases)(nil)
