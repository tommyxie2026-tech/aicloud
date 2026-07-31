package admission

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestApprovedHostedModelRequiresEvidenceAndPasses(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service := NewService(store)
	model := approvedModel(domain.DeploymentPublicAPI)
	evidence := approvedEvidence(model, now)
	if err := service.Append(context.Background(), evidence); err != nil {
		t.Fatalf("Append: %v", err)
	}
	decision, err := service.Check(context.Background(), model)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !decision.Allowed || decision.EvidenceID != evidence.ID {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestSelfHostedModelRequiresArtifactSignatureAndScan(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryStore())
	model := approvedModel(domain.DeploymentSelfHosted)
	evidence := approvedEvidence(model, now)
	evidence.ArtifactDigest = "sha256:artifact"
	if err := service.Append(context.Background(), evidence); err != nil {
		t.Fatalf("Append: %v", err)
	}
	decision, err := service.Check(context.Background(), model)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision.Allowed {
		t.Fatal("expected missing signature and scan evidence to deny admission")
	}
	if len(decision.Reasons) != 2 {
		t.Fatalf("unexpected reasons: %#v", decision.Reasons)
	}
}

func TestLatestRevocationOverridesPriorApproval(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service := NewService(store)
	model := approvedModel(domain.DeploymentPublicAPI)
	approved := approvedEvidence(model, now)
	if err := service.Append(context.Background(), approved); err != nil {
		t.Fatalf("Append approved: %v", err)
	}
	revoked := approved
	revoked.ID = "evidence-revoked"
	revoked.Status = StatusRevoked
	revoked.EvidenceDigest = "sha256:revoked"
	revoked.CreatedAt = now.Add(time.Minute)
	if err := service.Append(context.Background(), revoked); err != nil {
		t.Fatalf("Append revoked: %v", err)
	}
	decision, err := service.Check(context.Background(), model)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if decision.Allowed || len(decision.Reasons) != 1 || decision.Reasons[0] != "latest model evidence is revoked" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func approvedModel(mode domain.DeploymentMode) domain.Model {
	return domain.Model{
		ID: "model-1", Version: "v1", DeploymentMode: mode,
		ApprovalStatus: domain.ApprovalApproved,
	}
}

func approvedEvidence(model domain.Model, now time.Time) Evidence {
	return Evidence{
		ID: "evidence-approved", ModelID: model.ID, ModelVersion: model.Version,
		Status: StatusApproved, LicenseID: "license-1",
		LicenseTextRef: "https://licenses.example/license-1", SourceRef: "https://models.example/model-1",
		CommercialUseAllowed: true, HostedServiceAllowed: true,
		Reviewer: "reviewer", ReviewedAt: &now, EvidenceDigest: "sha256:evidence",
		CreatedAt: now,
	}
}
