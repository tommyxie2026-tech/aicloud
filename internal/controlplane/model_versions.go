package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (s *Service) WithModelVersions(repo domain.ModelVersionRepository) *Service {
	if s.router != nil {
		s.router.WithModelVersions(repo)
	}
	return s
}
