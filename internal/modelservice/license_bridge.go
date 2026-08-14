package modelservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var errLicenseEvidenceNotFound = errors.New("license evidence version not found")

func (s *Service) syncVersionedLicenseEvidence(ctx context.Context, model domain.Model) error {
	licenseRepo := s.LicenseEvidenceVersionRepository()
	if licenseRepo == nil {
		return nil
	}
	legacy := model.LicenseEvidence
	if model.ApprovalStatus != domain.ApprovalApproved || legacy.LicenseID == "" || legacy.LicenseTextRef == "" || legacy.Reviewer == "" || legacy.ReviewedAt == nil {
		return nil
	}
	if err := s.ensureStaticModelVersion(ctx, model); err != nil {
		return err
	}

	digest := legacyLicenseDigest(model)
	version := strings.TrimPrefix(digest, "sha256:")
	if len(version) > 16 {
		version = version[:16]
	}
	item := domain.LicenseEvidenceVersion{
		ID:                  "license-" + model.ID,
		Version:             version,
		ModelVersionID:      model.ID,
		LicenseID:           legacy.LicenseID,
		WeightAvailability:  permission(model.DeploymentMode == domain.DeploymentSelfHosted || model.DeploymentMode == domain.DeploymentLocal),
		CommercialUse:       permission(legacy.CommercialUseAllowed),
		HostedService:       permission(legacy.HostedServiceAllowed),
		Redistribution:      permission(legacy.RedistributionAllowed),
		DerivativeWorks:     domain.LicenseConditional,
		AttributionRequired: legacy.NoticeRequired,
		NoticeRequired:      legacy.NoticeRequired,
		EffectiveFrom:       legacy.ReviewedAt.UTC(),
		EvidenceRef:         legacy.LicenseTextRef,
		EvidenceDigest:      digest,
		Reviewer:            legacy.Reviewer,
		ApprovalState:       domain.LicenseApproved,
		CreatedAt:           legacy.ReviewedAt.UTC(),
	}
	if _, err := licenseRepo.Get(ctx, item.ID, item.Version); err == nil {
		return nil
	}
	if _, err := licenseRepo.Create(ctx, item); err != nil {
		return fmt.Errorf("create versioned license evidence: %w", err)
	}
	return nil
}

func (s *Service) ensureStaticModelVersion(ctx context.Context, model domain.Model) error {
	repo := s.ModelVersionRepository()
	if repo == nil {
		return nil
	}
	if _, err := repo.Get(ctx, model.ID); err == nil {
		return nil
	}
	item := domain.ModelVersionFromLegacy(model)
	if _, err := repo.Create(ctx, item); err != nil {
		// Memory legacy projections are read-only but already resolve from the
		// just-created legacy model. Recheck before treating Create as fatal.
		if _, getErr := repo.Get(ctx, model.ID); getErr == nil {
			return nil
		}
		return fmt.Errorf("ensure static model version: %w", err)
	}
	return nil
}

func legacyLicenseDigest(model domain.Model) string {
	legacy := model.LicenseEvidence
	parts := []string{
		model.ID, model.Version, legacy.LicenseID, legacy.LicenseTextRef,
		fmt.Sprintf("commercial=%t", legacy.CommercialUseAllowed),
		fmt.Sprintf("hosted=%t", legacy.HostedServiceAllowed),
		fmt.Sprintf("redistribution=%t", legacy.RedistributionAllowed),
		fmt.Sprintf("notice=%t", legacy.NoticeRequired), legacy.Reviewer,
	}
	if legacy.ReviewedAt != nil {
		parts = append(parts, legacy.ReviewedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func permission(allowed bool) domain.LicensePermission {
	if allowed {
		return domain.LicenseAllowed
	}
	return domain.LicenseForbidden
}

func isLicenseEvidenceNotFound(err error) bool {
	return errors.Is(err, errLicenseEvidenceNotFound)
}
