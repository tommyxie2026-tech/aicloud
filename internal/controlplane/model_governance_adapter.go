package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (a modelRepositoryAdapter) LicenseEvidenceVersionRepository() domain.LicenseEvidenceVersionRepository {
	if a.service == nil {
		return nil
	}
	return a.service.LicenseEvidenceVersionRepository()
}
