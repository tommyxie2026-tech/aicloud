package domain

import (
	"context"
	"time"
)

// ModelVersion contains catalog and governance evidence only. Mutable runtime
// state such as endpoint, pricing, health, quota, capacity, region and queue
// belongs to Deployment.
type ModelVersion struct {
	ID                         string          `json:"id"`
	Name                       string          `json:"name"`
	Version                    string          `json:"version"`
	Lifecycle                  ModelLifecycle  `json:"lifecycle"`
	Capabilities               []string        `json:"capabilities,omitempty"`
	EvaluationVersion          string          `json:"evaluationVersion,omitempty"`
	License                    string          `json:"license,omitempty"`
	LicenseEvidence            LicenseEvidence `json:"licenseEvidence,omitempty"`
	LicenseEvidenceVersionRef  string          `json:"licenseEvidenceVersionRef,omitempty"`
	Provenance                 ModelProvenance `json:"provenance,omitempty"`
	ArtifactDigest             string          `json:"artifactDigest,omitempty"`
	ApprovalStatus             ApprovalStatus  `json:"approvalStatus"`
	RiskLevel                  string          `json:"riskLevel,omitempty"`
	CreatedAt                  time.Time       `json:"createdAt"`
	UpdatedAt                  time.Time       `json:"updatedAt"`
}

type ModelVersionRepository interface {
	List(context.Context) ([]ModelVersion, error)
	Get(context.Context, string) (ModelVersion, error)
	Create(context.Context, ModelVersion) (ModelVersion, error)
	Update(context.Context, ModelVersion) (ModelVersion, error)
}
