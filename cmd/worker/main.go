package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := cfg.Temporal.Validate(); err != nil {
		log.Error("Temporal worker configuration invalid", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(nil, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !cfg.Temporal.Enabled {
		log.Info("aicloud worker idle", "temporalEnabled", false)
		<-ctx.Done()
		return
	}

	workerConfig := workflow.WorkerConfig{
		Enabled:     true,
		Address:     cfg.Temporal.Address,
		Namespace:   cfg.Temporal.Namespace,
		TaskQueue:   cfg.Temporal.TaskQueue,
		StopTimeout: time.Duration(cfg.Temporal.WorkerStopTimeoutSeconds) * time.Second,
	}
	log.Info(
		"aicloud Temporal worker starting",
		"address", workerConfig.Address,
		"namespace", workerConfig.Namespace,
		"taskQueue", workerConfig.TaskQueue,
	)
	if err := workflow.RunTemporalWorker(ctx, workerConfig, workflow.FailClosedLifecycleActivities{}); err != nil {
		log.Error("aicloud Temporal worker stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("aicloud Temporal worker stopped")
}
