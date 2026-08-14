package repository

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func (r *PostgresDeployments) PricingPolicies() domain.PricingPolicyRepository {
	if r == nil || r.db == nil {
		return nil
	}
	return NewPostgresPricingPolicies(r.db)
}
