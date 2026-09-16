package workflow

import (
	"context"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
)

func TestDurableExecutionActivityRunningNodeDelegatesLeaseRecoveryToWorker(t *testing.T) {
	activity, task, plan, runtime, worker := executionActivityFixture(t)
	lineage := activity.Lineage.(*executionLineageStub)
	lineage.execution = fixtureExecution(task, plan, time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC))
	runtime.record = fixtureRuntime(lineage.execution, plan, execution.NodeRunning)

	if err := activity.Execute(context.Background(), executionStepInput(task)); err != nil {
		t.Fatalf("Execute running-node recovery returned error: %v", err)
	}
	if worker.calls != 1 {
		t.Fatalf("running node must delegate lease recovery to worker, calls=%d", worker.calls)
	}
	if runtime.requeues != 0 {
		t.Fatalf("running node must not use FAILED requeue path, requeues=%d", runtime.requeues)
	}
}
