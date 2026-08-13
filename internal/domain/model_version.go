package domain

import (
	"context"
	"time"
)

type ModelVersion struct {
	ID                string
	Name              string
	Version           string
	Lifecycle         ModelLifecycle
	Capabilities      []string
	EvaluationVersion string
	License           string
	LicenseEvidence   LicenseEvidence
	Provenance        ModelProvenance
	ArtifactDigest    string
	ApprovalStatus    ApprovalStatus
	RiskLevel         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ModelVersionRepository interface {
	List(context.Context) ([]ModelVersion, error)
	Get(context.Context, string) (ModelVersion, error)
	Create(context.Context, ModelVersion) (ModelVersion, error)
	Update(context.Context, ModelVersion) (ModelVersion, error)
}
