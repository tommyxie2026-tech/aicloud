package workflow

import (
	"context"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func TestWorkerConfigValidation(t *testing.T) {
	if err := (WorkerConfig{}).Validate(); err != nil {
		t.Fatalf("disabled config rejected: %v", err)
	}

	valid := WorkerConfig{
		Enabled:     true,
		Address:     "localhost:7233",
		Namespace:   "default",
		TaskQueue:   "aicloud-task-v1",
		StopTimeout: 30 * time.Second,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*WorkerConfig)
	}{
		{name: "address", mutate: func(c *WorkerConfig) { c.Address = "" }},
		{name: "namespace", mutate: func(c *WorkerConfig) { c.Namespace = "" }},
		{name: "task queue", mutate: func(c *WorkerConfig) { c.TaskQueue = "" }},
		{name: "stop timeout", mutate: func(c *WorkerConfig) { c.StopTimeout = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := valid
			tc.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Fatal("invalid enabled worker config was accepted")
			}
		})
	}
}

func TestDisabledWorkerDoesNotDialTemporal(t *testing.T) {
	dialed := false
	dial := func(context.Context, client.Options) (client.Client, error) {
		dialed = true
		return nil, nil
	}
	if err := runTemporalWorker(context.Background(), WorkerConfig{Enabled: false}, nil, dial, nil); err != nil {
		t.Fatalf("disabled worker returned error: %v", err)
	}
	if dialed {
		t.Fatal("disabled worker dialed Temporal")
	}
}

func TestTemporalWorkerOptionsEnforceS3CSafetyContract(t *testing.T) {
	config := WorkerConfig{StopTimeout: 37 * time.Second}
	options := temporalWorkerOptions(config)
	if options.WorkerStopTimeout != config.StopTimeout {
		t.Fatalf("worker stop timeout=%s", options.WorkerStopTimeout)
	}
	if options.WorkflowPanicPolicy != worker.BlockWorkflow {
		t.Fatalf("workflow panic policy=%v", options.WorkflowPanicPolicy)
	}
	if !options.DisableRegistrationAliasing {
		t.Fatal("registration aliasing must be disabled")
	}
}
