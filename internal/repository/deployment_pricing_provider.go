package repository

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

// PricingPoliciesForDeployments exposes the pricing-policy repository that
// shares the same persistence boundary as a deployment repository. Memory
// deployments intentionally return nil unless a caller binds an explicit
// pricing repository, preserving lightweight development compatibility.
func PricingPoliciesForDeployments(deployments domain.DeploymentRepository) domain.PricingPolicyRepository {
	switch repo := deployments.(type) {
	case *PostgresDeployments:
		return NewPostgresPricingPolicies(repo.db)
	default:
		return nil
	}
}
