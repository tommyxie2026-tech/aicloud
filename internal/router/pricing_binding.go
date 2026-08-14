package router

import (
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var pricingPolicyBindings sync.Map

func (r *Router) WithPricingPolicies(repo domain.PricingPolicyRepository) *Router {
	if r != nil {
		if repo == nil {
			pricingPolicyBindings.Delete(r)
		} else {
			pricingPolicyBindings.Store(r, repo)
		}
	}
	return r
}

func (r *Router) pricingPolicyRepository() domain.PricingPolicyRepository {
	if r == nil {
		return nil
	}
	value, ok := pricingPolicyBindings.Load(r)
	if !ok {
		return nil
	}
	repo, _ := value.(domain.PricingPolicyRepository)
	return repo
}
