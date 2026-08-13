package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (s *Service) WithDeployments(repo domain.DeploymentRepository) *Service {
	if s != nil && s.router != nil {
		s.router.WithDeployments(repo)
		if s.models != nil {
			s.router.WithModelVersions(serviceModelVersions{models: s.models})
		}
	}
	return s
}
