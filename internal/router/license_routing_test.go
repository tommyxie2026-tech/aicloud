package router

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestDeploymentRoutingUsesExactApprovedLicenseEvidence(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	models := repository.NewMemoryModels(
		domain.Model{ID: "model-allowed", Version: "v1", Lifecycle: domain.ModelActive, ApprovalStatus: domain.ApprovalApproved, Capabilities: []string{"coding"}},
		domain.Model{ID: "model-forbidden", Version: "v1", Lifecycle: domain.ModelActive, ApprovalStatus: domain.ApprovalApproved, Capabilities: []string{"coding"}},
	)
	deployments := repository.NewMemoryDeployments(
		routableDeployment("deployment-allowed", "model-allowed", now),
		routableDeployment("deployment-forbidden", "model-forbidden", now),
	)
	licenses := repository.NewMemoryLicenseEvidenceVersions()
	allowed := testLicenseEvidence("license-allowed", "v1", "model-allowed", now)
	forbidden := testLicenseEvidence("license-forbidden", "v1", "model-forbidden", now)
	forbidden.CommercialUse = domain.LicenseForbidden
	for _, item := range []domain.LicenseEvidenceVersion{allowed, forbidden} {
		if _, err := licenses.Create(ctx, item); err != nil {
			t.Fatalf("create license evidence: %v", err)
		}
	}

	r := New(models, nil).
		WithDeployments(deployments).
		WithLicenseEvidenceVersions(licenses)
	r.now = func() time.Time { return now }
	decision, err := r.Plan(ctx, Request{TaskID: "task-license", RequiredCapabilities: []string{"coding"}})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.DeploymentID != "deployment-allowed" {
		t.Fatalf("selected deployment = %s, want deployment-allowed", decision.Selected.DeploymentID)
	}
	if !strings.Contains(decision.EvidenceVersion, "license:"+allowed.Ref()) {
		t.Fatalf("route evidence %q does not retain exact license version %s", decision.EvidenceVersion, allowed.Ref())
	}
	if !strings.Contains(decision.Reason, allowed.Ref()) {
		t.Fatalf("route reason %q does not identify license evidence", decision.Reason)
	}
}

func TestDeploymentRoutingRejectsFreeTextLicenseWithoutVersionedEvidence(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	model := domain.Model{
		ID: "model-free-text", Version: "v1", Lifecycle: domain.ModelActive,
		ApprovalStatus: domain.ApprovalApproved, License: "looks-open",
	}
	models := repository.NewMemoryModels(model)
	deployments := repository.NewMemoryDeployments(routableDeployment("deployment-free-text", model.ID, now))
	r := New(models, nil).
		WithDeployments(deployments).
		WithLicenseEvidenceVersions(repository.NewMemoryLicenseEvidenceVersions())
	r.now = func() time.Time { return now }

	if _, err := r.Plan(ctx, Request{TaskID: "task-free-text"}); err == nil {
		t.Fatal("free-text license without approved versioned evidence must not be routable")
	}
}

func routableDeployment(id, modelID string, now time.Time) domain.Deployment {
	return domain.Deployment{
		ID: id, ModelVersionID: modelID, ModelID: modelID, ModelVersion: "v1",
		Mode: domain.DeploymentPublicAPI, Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now, CapacityAvailable: 10, QuotaRemaining: 10,
	}
}

func testLicenseEvidence(id, version, modelID string, now time.Time) domain.LicenseEvidenceVersion {
	return domain.LicenseEvidenceVersion{
		ID: id, Version: version, ModelVersionID: modelID, LicenseID: "license-test",
		WeightAvailability: domain.LicenseAllowed, CommercialUse: domain.LicenseAllowed,
		HostedService: domain.LicenseAllowed, Redistribution: domain.LicenseForbidden,
		DerivativeWorks: domain.LicenseConditional, EffectiveFrom: now.Add(-time.Hour),
		EvidenceRef: "https://example.test/" + id, EvidenceDigest: "sha256:" + id,
		Reviewer: "reviewer", ApprovalState: domain.LicenseApproved, CreatedAt: now.Add(-time.Hour),
	}
}
