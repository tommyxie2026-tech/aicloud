package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

type deploymentPricingProvider interface {
	PricingPolicies() domain.PricingPolicyRepository
}

func (s *Service) WithDeployments(repo domain.DeploymentRepository) *Service {
	if s != nil && s.router != nil {
		s.router.WithDeployments(repo)
		if s.models != nil {
			s.router.WithModelVersions(serviceModelVersions{models: s.models})
			s.router.WithLicenseEvidenceVersions(s.models.LicenseEvidenceVersionRepository())
		}
		if provider, ok := repo.(deploymentPricingProvider); ok {
			s.router.WithPricingPolicies(provider.PricingPolicies())
		}
	}
	return s
}
