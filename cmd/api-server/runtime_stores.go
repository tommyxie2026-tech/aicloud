package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/db/migrations"
	"github.com/tommyxie2026-tech/aicloud/internal/admission"
	"github.com/tommyxie2026-tech/aicloud/internal/circuitbreaker"
	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/evaluation"
	"github.com/tommyxie2026-tech/aicloud/internal/modelruntime"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/tenantrepo"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/model/mock"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

type runtimeStores struct {
	models      domain.ModelRepository
	tasks       domain.TaskRepository
	routes      domain.RouteDecisionRepository
	costs       domain.CostEventRepository
	traces      tracepkg.Store
	evaluations evaluation.Store
	admissions  admission.Store
	close       func()
}

func buildRuntimeStores(ctx context.Context, cfg config.Config) (runtimeStores, error) {
	if strings.EqualFold(cfg.RepositoryMode, "postgres") {
		repos, err := repository.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return runtimeStores{}, err
		}
		if cfg.RunMigrations {
			if err := migrations.Run(ctx, repos.DB); err != nil {
				_ = repos.DB.Close()
				return runtimeStores{}, err
			}
		}
		ownership := tenantrepo.NewPostgresOwnershipStore(repos.DB)
		tasks := tenantrepo.NewScopedTasks(repos.Tasks, ownership)
		routes := tenantrepo.NewScopedRouteDecisions(repos.RouteDecisions, tasks)
		costs := tenantrepo.NewScopedCostEvents(repos.CostEvents, tasks)
		return runtimeStores{
			models: repos.Models, tasks: tasks, routes: routes, costs: costs,
			traces: repository.NewPostgresTraceStore(repos.DB),
			evaluations: repository.NewPostgresEvaluationStore(repos.DB),
			admissions: repository.NewPostgresAdmissionStore(repos.DB),
			close: func() { _ = repos.DB.Close() },
		}, nil
	}
	ownership := tenantrepo.NewMemoryOwnershipStore()
	tasks := tenantrepo.NewScopedTasks(repository.NewMemoryTasks(), ownership)
	routes := tenantrepo.NewScopedRouteDecisions(repository.NewMemoryRouteDecisions(), tasks)
	costs := tenantrepo.NewScopedCostEvents(repository.NewMemoryCostEvents(), tasks)
	return runtimeStores{
		models: repository.NewMemoryModels(), tasks: tasks, routes: routes, costs: costs,
		traces: tracepkg.NewMemoryStore(), evaluations: evaluation.NewMemoryStore(),
		admissions: admission.NewMemoryStore(), close: func() {},
	}, nil
}

func initializeModelRuntime(ctx context.Context, stores runtimeStores, models *modelservice.Service, cfg config.Config, configured provider.ModelProvider) (*admission.Service, *modelruntime.Executor, error) {
	admissionService := admission.NewService(stores.admissions)
	providerRegistry := modelruntime.NewMemoryProviderRegistry()

	mockModel, mockEvidence := mockRuntimeRecords(time.Now().UTC())
	if err := upsertModel(ctx, models, mockModel); err != nil {
		return nil, nil, err
	}
	if err := appendEvidenceOnce(ctx, stores.admissions, admissionService, mockEvidence); err != nil {
		return nil, nil, err
	}
	if err := providerRegistry.Put(ctx, mockModel.ID, mock.NewProvider()); err != nil {
		return nil, nil, err
	}

	if configured != nil {
		modelID := providerModelID(cfg.Provider.Name, cfg.Provider.DefaultModel, cfg.Provider.ModelVersion)
		if err := providerRegistry.Put(ctx, modelID, configured); err != nil {
			return nil, nil, err
		}
		evidence := configuredProviderEvidence(modelID, cfg.Provider, time.Now().UTC())
		if err := appendEvidenceOnce(ctx, stores.admissions, admissionService, evidence); err != nil {
			return nil, nil, err
		}
	}

	breaker := circuitbreaker.New(circuitbreaker.NewMemoryStore(), 3, time.Minute)
	ledger := cost.New(stores.costs)
	runtime := modelruntime.NewExecutor(providerRegistry, stores.models, breaker, ledger, stores.traces)
	return admissionService, runtime, nil
}

func upsertModel(ctx context.Context, models *modelservice.Service, model domain.Model) error {
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

func appendEvidenceOnce(ctx context.Context, store admission.Store, service *admission.Service, evidence admission.Evidence) error {
	if _, err := store.Get(ctx, evidence.ID); err == nil {
		return nil
	}
	return service.Append(ctx, evidence)
}

func mockRuntimeRecords(now time.Time) (domain.Model, admission.Evidence) {
	model := domain.Model{
		ID: "mock-model-v1", Name: "Mock Model", Version: "v1", Provider: "mock",
		DeploymentMode: domain.DeploymentLocal, Lifecycle: domain.ModelActive,
		Capabilities: []string{"structured-output", "json-schema", "chinese", "local-deployment"},
		Pricing: domain.PricingProfile{Currency: "USD"}, Health: domain.HealthHealthy,
		HealthCheckedAt: &now, QuotaRemaining: -1, CapacityAvailable: -1,
		ServiceTiers: []domain.ServiceTier{domain.TierStandard},
		InferenceEfforts: []domain.InferenceEffort{domain.EffortLow},
		EvaluationVersion: "mock-golden-v1", License: "internal",
		ApprovalStatus: domain.ApprovalApproved, RiskLevel: "low",
		CreatedAt: now, UpdatedAt: now,
	}
	evidence := admission.Evidence{
		ID: "admission-mock-model-v1", ModelID: model.ID, ModelVersion: model.Version,
		Status: admission.StatusApproved, LicenseID: "internal",
		LicenseTextRef: "internal://licenses/aicloud-mock-v1", SourceRef: "internal://model/mock",
		CommercialUseAllowed: true, HostedServiceAllowed: true, RedistributionAllowed: false,
		ArtifactDigest: "sha256:mock-deterministic-v1", ArtifactSignature: "internal-signature:mock-v1",
		SecurityScanRef: "internal://security-scans/mock-v1", Reviewer: "aicloud-maintainers",
		ReviewedAt: &now, EvidenceDigest: evidenceDigest(model.ID, model.Version, "internal", "mock-v1"),
		CreatedAt: now,
	}
	return model, evidence
}

func configuredProviderEvidence(modelID string, cfg config.ProviderConfig, now time.Time) admission.Evidence {
	status := admission.StatusCollected
	var reviewedAt *time.Time
	if cfg.Approved {
		status = admission.StatusApproved
		reviewedAt = &now
	}
	return admission.Evidence{
		ID: "admission-" + shortDigest(modelID+"|"+cfg.LicenseID+"|"+cfg.LicenseTextRef),
		ModelID: modelID, ModelVersion: cfg.ModelVersion, Status: status,
		LicenseID: cfg.LicenseID, LicenseTextRef: cfg.LicenseTextRef, SourceRef: cfg.Endpoint,
		CommercialUseAllowed: cfg.Approved, HostedServiceAllowed: cfg.Approved,
		Reviewer: cfg.EvidenceReviewer, ReviewedAt: reviewedAt,
		EvidenceDigest: evidenceDigest(modelID, cfg.ModelVersion, cfg.LicenseID, cfg.LicenseTextRef),
		CreatedAt: now,
	}
}

func evidenceDigest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func shortDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func validateApprovedProviderEvidence(cfg config.ProviderConfig) error {
	if cfg.Approved && (cfg.LicenseTextRef == "" || cfg.EvidenceReviewer == "") {
		return fmt.Errorf("approved provider requires license text reference and evidence reviewer")
	}
	return nil
}
