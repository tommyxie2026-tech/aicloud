package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (s *Service) WithLicenseEvidenceVersions(repo domain.LicenseEvidenceVersionRepository) *Service {
	if s != nil && s.router != nil {
		s.router.WithLicenseEvidenceVersions(repo)
	}
	return s
}
