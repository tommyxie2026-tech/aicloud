package main

import (
	"context"
	"errors"
	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/httpapi"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/telemetry"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()
	log := logging.New(cfg.LogLevel)
	models := repository.NewMemoryModels(domain.Model{ID: "mock", Name: "Mock Model", Provider: "mock", Capabilities: []string{"structured-output"}, License: "internal"})
	control := controlplane.New(modelservice.New(models), repository.NewMemoryTasks(), workflow.NoopEngine{})
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: httpapi.New(control, log).Handler(), ReadHeaderTimeout: 5 * time.Second}
	telemetryProvider := telemetry.NoopProvider{}
	go func() {
		log.Info("api server started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	_ = telemetryProvider.Shutdown()
}
