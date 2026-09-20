package repository

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

type LicenseEvidenceRepositoryProvider interface {
	LicenseEvidenceVersionRepository() domain.LicenseEvidenceVersionRepository
}

func (r *PostgresModels) LicenseEvidenceVersionRepository() domain.LicenseEvidenceVersionRepository {
	if r == nil {
		return nil
	}
	return NewPostgresLicenseEvidenceVersions(r.db)
}

func (r *MemoryModels) LicenseEvidenceVersionRepository() domain.LicenseEvidenceVersionRepository {
	if r == nil {
		return nil
	}
	if r.licenseEvidence == nil {
		r.licenseEvidence = NewMemoryLicenseEvidenceVersions()
	}
	return r.licenseEvidence
}
