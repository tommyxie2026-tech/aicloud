package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tommyxie2026-tech/aicloud/db/migrations"
	"github.com/tommyxie2026-tech/aicloud/internal/approval"
	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/credentials"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/httpapi"
	"github.com/tommyxie2026-tech/aicloud/internal/logging"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/policy"
	"github.com/tommyxie2026-tech/aicloud/internal/providerfactory"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/sandbox"
	"github.com/tommyxie2026-tech/aicloud/internal/telemetry"
	"github.com/tommyxie2026-tech/aicloud/internal/toolgateway"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func main() {
	cfg := config.Load()
	log := logging.New(cfg.LogLevel)
	ctx := context.Background()

	modelRepo, taskRepo, routeRepo, costRepo, closeStore, err := buildRepositories(ctx, cfg)
	if err != nil {
		log.Error("repository initialization failed", "error", err)
		os.Exit(1)
	}
	defer closeStore()

	models := modelservice.New(modelRepo)
	if cfg.Provider.Enabled {
		adapter, err := providerfactory.BuildOpenAICompatible(cfg.Provider)
		if err != nil {
			log.Error("provider initialization failed", "error", err)
			os.Exit(1)
		}
		if err := registerConfiguredProvider(ctx, models, adapter, cfg.Provider, log); err != nil {
			log.Error("provider registration failed", "error", err)
			os.Exit(1)
		}
	}

	toolService, auditStore := buildSecureTools()
	control := controlplane.New(models, taskRepo, workflow.NoopEngine{}).
		WithGovernance(routeRepo, costRepo).
		WithSecureTools(toolService, auditStore)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(control, log).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	telemetryProvider := telemetry.NoopProvider{}
	go func() {
		log.Info("api server started", "addr", cfg.HTTPAddr, "repositoryMode", cfg.RepositoryMode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	_ = telemetryProvider.Shutdown()
}

func buildRepositories(ctx context.Context, cfg config.Config) (
	domain.ModelRepository,
	domain.TaskRepository,
	domain.RouteDecisionRepository,
	domain.CostEventRepository,
	func(),
	error,
) {
	if strings.EqualFold(cfg.RepositoryMode, "postgres") {
		repos, err := repository.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, nil, nil, nil, func() {}, err
		}
		if cfg.RunMigrations {
			if err := migrations.Run(ctx, repos.DB); err != nil {
				_ = repos.DB.Close()
				return nil, nil, nil, nil, func() {}, err
			}
		}
		return repos.Models, repos.Tasks, repos.RouteDecisions, repos.CostEvents, func() { _ = repos.DB.Close() }, nil
	}

	now := time.Now().UTC()
	mock := domain.Model{
		ID:                "mock-model-v1",
		Name:              "Mock Model",
		Version:           "v1",
		Provider:          "mock",
		DeploymentMode:    domain.DeploymentLocal,
		Lifecycle:         domain.ModelActive,
		Capabilities:      []string{"structured-output", "coding"},
		Pricing:           domain.PricingProfile{Currency: "USD"},
		Health:            domain.HealthHealthy,
		HealthCheckedAt:   &now,
		QuotaRemaining:    -1,
		CapacityAvailable: -1,
		ServiceTiers:      []domain.ServiceTier{domain.TierStandard},
		InferenceEfforts:  []domain.InferenceEffort{domain.EffortLow},
		License:           "internal",
		ApprovalStatus:    domain.ApprovalApproved,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return repository.NewMemoryModels(mock), repository.NewMemoryTasks(), repository.NewMemoryRouteDecisions(), repository.NewMemoryCostEvents(), func() {}, nil
}

func buildSecureTools() (*toolgateway.Service, *audit.MemoryStore) {
	registry := toolgateway.NewMemoryRegistry(
		toolgateway.Definition{
			ID: "repo-inspect", Version: "v1", Image: "alpine/git:2.45.2",
			Command: []string{"git", "status", "--short"}, RiskLevel: "low",
			Permission: "repository:read", CredentialTTL: 2 * time.Minute,
			CPU: "250m", Memory: "256Mi", Timeout: 2 * time.Minute,
			NetworkMode: sandbox.NetworkDenyAll, WorkspaceWrite: false,
		},
		toolgateway.Definition{
			ID: "workspace-command", Version: "v1", Image: "alpine:3.20",
			Command: []string{"/bin/echo"}, RiskLevel: "high",
			Permission: "workspace:write", CredentialTTL: 2 * time.Minute,
			CPU: "500m", Memory: "512Mi", Timeout: 2 * time.Minute,
			NetworkMode: sandbox.NetworkDenyAll, WorkspaceWrite: true,
		},
	)
	policyEngine := policy.StaticEngine{Version: "builtin-tool-policy-v1", Rules: []policy.Rule{
		{Name: "allow-repo-inspect", Subject: "*", Action: "inspect", Resource: "repo-inspect", Allowed: true, Reason: "read-only repository inspection"},
		{Name: "approve-workspace-command", Subject: "*", Action: "execute", Resource: "workspace-command", Allowed: true, RequireApproval: true, Reason: "workspace modifications require approval"},
	}}
	auditStore := audit.NewMemoryStore()
	service := toolgateway.NewService(
		registry,
		policyEngine,
		approval.NewMemoryStore(),
		credentials.NewMemoryBroker(),
		sandbox.NewPlanningExecutor(),
		auditStore,
	)
	return service, auditStore
}

func registerConfiguredProvider(ctx context.Context, models *modelservice.Service, adapter provider.ModelProvider, cfg config.ProviderConfig, log *slog.Logger) error {
	if adapter == nil {
		return nil
	}
	now := time.Now().UTC()
	healthStatus := domain.HealthUnknown
	lifecycle := domain.ModelDraft
	health, healthErr := adapter.Health(ctx)
	if healthErr != nil || health == nil || !health.Available {
		healthStatus = domain.HealthUnhealthy
		if cfg.Approved {
			lifecycle = domain.ModelDegraded
		}
		if log != nil {
			log.Warn("provider health check failed", "provider", cfg.Name, "error", healthErr)
		}
	} else {
		healthStatus = domain.HealthHealthy
		if cfg.Approved {
			lifecycle = domain.ModelActive
		}
	}
	approvalStatus := domain.ApprovalDiscovered
	if cfg.Approved {
		approvalStatus = domain.ApprovalApproved
	}
	deployment := domain.DeploymentPublicAPI
	switch adapter.Type() {
	case provider.ProviderTypePrivate:
		deployment = domain.DeploymentPrivateEndpoint
	case provider.ProviderTypeLocal:
		deployment = domain.DeploymentLocal
	}
	capabilities := capabilityNames(adapter.Capabilities())
	model := domain.Model{
		ID:             providerModelID(cfg.Name, cfg.DefaultModel, cfg.ModelVersion),
		Name:           cfg.DefaultModel,
		Version:        cfg.ModelVersion,
		Provider:       cfg.Name,
		Endpoint:       cfg.Endpoint,
		DeploymentMode: deployment,
		Lifecycle:      lifecycle,
		Capabilities:   capabilities,
		Pricing: domain.PricingProfile{
			Currency:         cfg.Currency,
			InputPerMillion:  cfg.InputPerMillion,
			OutputPerMillion: cfg.OutputPerMillion,
		},
		Health:            healthStatus,
		HealthCheckedAt:   &now,
		QuotaRemaining:    -1,
		CapacityAvailable: -1,
		ServiceTiers:      []domain.ServiceTier{domain.TierStandard, domain.TierPriority, domain.TierBatch},
		InferenceEfforts:  []domain.InferenceEffort{domain.EffortLow, domain.EffortMedium, domain.EffortHigh},
		License:           cfg.LicenseID,
		LicenseEvidence: domain.LicenseEvidence{
			LicenseID:            cfg.LicenseID,
			CommercialUseAllowed: cfg.Approved,
			HostedServiceAllowed: cfg.Approved,
			Reviewer:             "runtime-configuration",
			ReviewedAt:           reviewedAt(cfg.Approved, now),
		},
		Provenance:     domain.ModelProvenance{Source: cfg.Endpoint},
		ApprovalStatus: approvalStatus,
		DataResidency:  cfg.DataResidency,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	current, err := models.Get(ctx, model.ID)
	if errors.Is(err, repository.ErrNotFound) {
		_, err = models.Create(ctx, model)
		return err
	}
	if err != nil {
		return err
	}
	model.CreatedAt = current.CreatedAt
	_, err = models.Update(ctx, model)
	return err
}

func providerModelID(providerName, modelName, version string) string {
	value := strings.ToLower(strings.Join([]string{providerName, modelName, version}, "-"))
	replacer := strings.NewReplacer(" ", "-", ":", "-", "@", "-", "/", "-")
	return replacer.Replace(value)
}

func reviewedAt(approved bool, value time.Time) *time.Time {
	if !approved {
		return nil
	}
	return &value
}

func capabilityNames(capabilities provider.ProviderCapabilities) []string {
	items := make([]string, 0)
	add := func(enabled bool, name string) {
		if enabled {
			items = append(items, name)
		}
	}
	add(capabilities.SupportsStructuredOutput, "structured-output")
	add(capabilities.SupportsJSONSchema, "json-schema")
	add(capabilities.SupportsStreaming, "streaming")
	add(capabilities.SupportsToolUse, "tool-use")
	add(capabilities.SupportsVision, "vision")
	add(capabilities.SupportsLongContext, "long-context")
	add(capabilities.SupportsChinese, "chinese")
	add(capabilities.SupportsCodeGeneration, "coding")
	add(capabilities.SupportsLocalDeployment, "local-deployment")
	for _, task := range capabilities.RecommendedTasks {
		items = append(items, fmt.Sprintf("task:%s", task))
	}
	return items
}
