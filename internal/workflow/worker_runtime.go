package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type WorkerConfig struct {
	Enabled     bool
	Address     string
	Namespace   string
	TaskQueue   string
	StopTimeout time.Duration
}

func (c WorkerConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Address) == "" {
		return fmt.Errorf("Temporal address is required")
	}
	if strings.TrimSpace(c.Namespace) == "" {
		return fmt.Errorf("Temporal namespace is required")
	}
	if strings.TrimSpace(c.TaskQueue) == "" {
		return fmt.Errorf("Temporal task queue is required")
	}
	if c.StopTimeout <= 0 {
		return fmt.Errorf("Temporal worker stop timeout must be greater than zero")
	}
	return nil
}

type temporalDialFunc func(context.Context, client.Options) (client.Client, error)
type temporalWorkerFactory func(client.Client, string, worker.Options) worker.Worker

func RunTemporalWorker(ctx context.Context, config WorkerConfig, activities LifecycleActivities) error {
	return runTemporalWorker(ctx, config, activities, dialTemporalClient, worker.New)
}

func runTemporalWorker(
	ctx context.Context,
	config WorkerConfig,
	activities LifecycleActivities,
	dial temporalDialFunc,
	newWorker temporalWorkerFactory,
) error {
	if err := config.Validate(); err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	if activities == nil {
		return fmt.Errorf("lifecycle activities are required when Temporal is enabled")
	}
	if dial == nil || newWorker == nil {
		return fmt.Errorf("Temporal worker dependencies are required")
	}

	temporalClient, err := dial(ctx, client.Options{
		HostPort:  strings.TrimSpace(config.Address),
		Namespace: strings.TrimSpace(config.Namespace),
	})
	if err != nil {
		return fmt.Errorf("dial Temporal: %w", err)
	}
	defer temporalClient.Close()

	temporalWorker := newWorker(temporalClient, strings.TrimSpace(config.TaskQueue), worker.Options{
		WorkerStopTimeout:           config.StopTimeout,
		WorkflowPanicPolicy:         worker.BlockWorkflow,
		DisableRegistrationAliasing: true,
	})
	if temporalWorker == nil {
		return fmt.Errorf("Temporal worker factory returned nil worker")
	}
	if err := RegisterLifecycle(temporalWorker, activities); err != nil {
		return err
	}
	if err := temporalWorker.Start(); err != nil {
		return fmt.Errorf("start Temporal worker: %w", err)
	}

	<-ctx.Done()
	temporalWorker.Stop()
	return nil
}

func dialTemporalClient(ctx context.Context, options client.Options) (client.Client, error) {
	return client.DialContext(ctx, options)
}
