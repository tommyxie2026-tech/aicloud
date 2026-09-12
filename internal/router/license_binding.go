package router

import (
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var licenseEvidenceBindings sync.Map

func (r *Router) WithLicenseEvidenceVersions(repo domain.LicenseEvidenceVersionRepository) *Router {
	if r != nil {
		if repo == nil {
			licenseEvidenceBindings.Delete(r)
		} else {
			licenseEvidenceBindings.Store(r, repo)
		}
	}
	return r
}

func (r *Router) licenseEvidenceRepository() domain.LicenseEvidenceVersionRepository {
	if r == nil {
		return nil
	}
	value, ok := licenseEvidenceBindings.Load(r)
	if !ok {
		return nil
	}
	repo, _ := value.(domain.LicenseEvidenceVersionRepository)
	return repo
}
