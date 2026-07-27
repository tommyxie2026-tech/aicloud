package admission

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type Status string

const (
	StatusCollected  Status = "evidence-collected"
	StatusReviewed   Status = "reviewed"
	StatusApproved   Status = "approved"
	StatusRestricted Status = "restricted"
	StatusRejected   Status = "rejected"
	StatusRevoked    Status = "revoked"
)

type Evidence struct {
	ID                    string     `json:"id"`
	ModelID               string     `json:"modelId"`
	ModelVersion          string     `json:"modelVersion"`
	Status                Status     `json:"status"`
	LicenseID             string     `json:"licenseId"`
	LicenseTextRef        string     `json:"licenseTextRef"`
	SourceRef             string     `json:"sourceRef"`
	UpstreamModels        []string   `json:"upstreamModels,omitempty"`
	DatasetRefs           []string   `json:"datasetRefs,omitempty"`
	CommercialUseAllowed  bool       `json:"commercialUseAllowed"`
	HostedServiceAllowed  bool       `json:"hostedServiceAllowed"`
	RedistributionAllowed bool       `json:"redistributionAllowed"`
	NoticeRequired        bool       `json:"noticeRequired"`
	NoticeRef             string     `json:"noticeRef,omitempty"`
	ArtifactDigest        string     `json:"artifactDigest,omitempty"`
	ArtifactSignature     string     `json:"artifactSignature,omitempty"`
	SecurityScanRef       string     `json:"securityScanRef,omitempty"`
	Reviewer              string     `json:"reviewer"`
	ReviewedAt            *time.Time `json:"reviewedAt,omitempty"`
	EvidenceDigest        string     `json:"evidenceDigest"`
	CreatedAt             time.Time  `json:"createdAt"`
}

type Decision struct {
	Allowed    bool     `json:"allowed"`
	EvidenceID string   `json:"evidenceId,omitempty"`
	Reasons    []string `json:"reasons,omitempty"`
}

type Store interface {
	Append(context.Context, Evidence) error
	Get(context.Context, string) (Evidence, error)
	ListByModel(context.Context, string, string) ([]Evidence, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string]Evidence
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[string]Evidence)} }

func (s *MemoryStore) Append(_ context.Context, evidence Evidence) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[evidence.ID]; exists {
		return fmt.Errorf("admission evidence already exists")
	}
	s.records[evidence.ID] = cloneEvidence(evidence)
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id string) (Evidence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	evidence, ok := s.records[id]
	if !ok {
		return Evidence{}, fmt.Errorf("admission evidence not found")
	}
	return cloneEvidence(evidence), nil
}

func (s *MemoryStore) ListByModel(_ context.Context, modelID, modelVersion string) ([]Evidence, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Evidence, 0)
	for _, evidence := range s.records {
		if evidence.ModelID == modelID && evidence.ModelVersion == modelVersion {
			items = append(items, cloneEvidence(evidence))
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) Append(ctx context.Context, evidence Evidence) error {
	if evidence.ID == "" || evidence.ModelID == "" || evidence.ModelVersion == "" || evidence.EvidenceDigest == "" {
		return fmt.Errorf("evidence ID, model identity and evidence digest are required")
	}
	if evidence.CreatedAt.IsZero() {
		evidence.CreatedAt = time.Now().UTC()
	}
	return s.store.Append(ctx, evidence)
}

func (s *Service) List(ctx context.Context, modelID, modelVersion string) ([]Evidence, error) {
	return s.store.ListByModel(ctx, modelID, modelVersion)
}

func (s *Service) Check(ctx context.Context, model domain.Model) (Decision, error) {
	if model.ApprovalStatus != domain.ApprovalApproved {
		return Decision{Reasons: []string{"model approval status is not approved"}}, nil
	}
	records, err := s.store.ListByModel(ctx, model.ID, model.Version)
	if err != nil {
		return Decision{}, err
	}
	for index := len(records) - 1; index >= 0; index-- {
		evidence := records[index]
		if evidence.Status == StatusRevoked {
			return Decision{EvidenceID: evidence.ID, Reasons: []string{"latest model evidence is revoked"}}, nil
		}
		if evidence.Status != StatusApproved && evidence.Status != StatusRestricted {
			continue
		}
		reasons := validateEvidence(model, evidence)
		return Decision{Allowed: len(reasons) == 0, EvidenceID: evidence.ID, Reasons: reasons}, nil
	}
	return Decision{Reasons: []string{"no approved evidence record exists for the immutable model version"}}, nil
}

func validateEvidence(model domain.Model, evidence Evidence) []string {
	reasons := make([]string, 0)
	require := func(value, reason string) {
		if value == "" {
			reasons = append(reasons, reason)
		}
	}
	require(evidence.LicenseID, "license identifier is missing")
	require(evidence.LicenseTextRef, "authoritative license text reference is missing")
	require(evidence.SourceRef, "model source reference is missing")
	require(evidence.Reviewer, "evidence reviewer is missing")
	require(evidence.EvidenceDigest, "evidence digest is missing")
	if evidence.ReviewedAt == nil {
		reasons = append(reasons, "evidence review time is missing")
	}
	if !evidence.CommercialUseAllowed {
		reasons = append(reasons, "commercial use is not allowed")
	}
	if model.DeploymentMode == domain.DeploymentPublicAPI || model.DeploymentMode == domain.DeploymentEnterpriseAPI || model.DeploymentMode == domain.DeploymentPrivateEndpoint {
		if !evidence.HostedServiceAllowed {
			reasons = append(reasons, "hosted-service use is not allowed")
		}
	}
	if model.DeploymentMode == domain.DeploymentSelfHosted || model.DeploymentMode == domain.DeploymentLocal {
		require(evidence.ArtifactDigest, "self-hosted artifact digest is missing")
		require(evidence.ArtifactSignature, "self-hosted artifact signature is missing")
		require(evidence.SecurityScanRef, "self-hosted security scan evidence is missing")
	}
	if evidence.NoticeRequired && evidence.NoticeRef == "" {
		reasons = append(reasons, "required attribution or notice evidence is missing")
	}
	return reasons
}

func cloneEvidence(evidence Evidence) Evidence {
	evidence.UpstreamModels = append([]string(nil), evidence.UpstreamModels...)
	evidence.DatasetRefs = append([]string(nil), evidence.DatasetRefs...)
	return evidence
}
