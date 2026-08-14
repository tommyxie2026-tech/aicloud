package main

import (
	"context"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if !cfg.Temporal.Enabled {
		log.Info("aicloud worker idle", "temporalEnabled", false)
		<-ctx.Done()
		return
	}

	security := config.LoadTemporalSecurity()
	if err := security.Validate(); err != nil {
		log.Error("Temporal transport configuration invalid", "error", err)
		os.Exit(1)
	}
	tlsConfig, err := security.TLSConfig()
	if err != nil {
		log.Error("Temporal TLS configuration failed", "error", err)
		os.Exit(1)
	}

	workerConfig := workflow.WorkerConfig{
		Enabled:     true,
		Address:     cfg.Temporal.Address,
		Namespace:   cfg.Temporal.Namespace,
		TaskQueue:   cfg.Temporal.TaskQueue,
		StopTimeout: time.Duration(cfg.Temporal.WorkerStopTimeoutSeconds) * time.Second,
		TLSConfig:   tlsConfig,
	}
	log.Info(
		"aicloud Temporal worker starting",
		"address", workerConfig.Address,
		"namespace", workerConfig.Namespace,
		"taskQueue", workerConfig.TaskQueue,
		"tlsEnabled", security.TLSEnabled,
		"mtlsEnabled", security.CertFile != "" && security.KeyFile != "",
	)
	if err := workflow.RunTemporalWorker(ctx, workerConfig, workflow.FailClosedLifecycleActivities{}); err != nil {
		log.Error("aicloud Temporal worker stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("aicloud Temporal worker stopped")
}
