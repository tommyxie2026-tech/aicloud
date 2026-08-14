package controlplane

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (s *Service) WithPricingPolicies(repo domain.PricingPolicyRepository) *Service {
	if s != nil && s.router != nil {
		s.router.WithPricingPolicies(repo)
	}
	return s
}
