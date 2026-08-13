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

	stores, err := buildRuntimeStores(ctx, cfg)
	if err != nil {
		log.Error("repository initialization failed", "error", err)
		os.Exit(1)
	}
	defer stores.close()

	models := modelservice.New(stores.models)
	var configuredProvider provider.ModelProvider
	if cfg.Provider.Enabled {
		if err := validateApprovedProviderEvidence(cfg.Provider); err != nil {
			log.Error("provider evidence configuration failed", "error", err)
			os.Exit(1)
		}
		configuredProvider, err = providerfactory.BuildOpenAICompatible(cfg.Provider)
		if err != nil {
			log.Error("provider initialization failed", "error", err)
			os.Exit(1)
		}
		if err := registerConfiguredProvider(ctx, models, configuredProvider, cfg.Provider, log); err != nil {
			log.Error("provider registration failed", "error", err)
			os.Exit(1)
		}
	}

	admissionService, modelRuntime, err := initializeModelRuntime(ctx, stores, models, cfg, configuredProvider)
	if err != nil {
		log.Error("model runtime initialization failed", "error", err)
		os.Exit(1)
	}
	toolService, auditStore := buildSecureTools()
	control := controlplane.New(models, stores.tasks, workflow.NoopEngine{}).
		WithGovernance(stores.routes, stores.costs).
		WithDeployments(stores.deployments).
		WithSecureTools(toolService, auditStore).
		WithEvidence(stores.traces, stores.evaluations, admissionService, modelRuntime)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.WithTenantScope(httpapi.New(control, log).FullHandler()),
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
			LicenseTextRef:       cfg.LicenseTextRef,
			CommercialUseAllowed: cfg.Approved,
			HostedServiceAllowed: cfg.Approved,
			Reviewer:             cfg.EvidenceReviewer,
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
