package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type MemoryPricingPolicies struct {
	mu    sync.RWMutex
	items map[string]domain.PricingPolicy
}

func NewMemoryPricingPolicies() *MemoryPricingPolicies {
	return &MemoryPricingPolicies{items: make(map[string]domain.PricingPolicy)}
}

func (r *MemoryPricingPolicies) Create(_ context.Context, item domain.PricingPolicy) (domain.PricingPolicy, error) {
	if err := item.Validate(); err != nil {
		return domain.PricingPolicy{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.Ref()]; ok {
		return domain.PricingPolicy{}, errors.New("pricing policy version already exists")
	}
	r.items[item.Ref()] = item
	return item, nil
}

func (r *MemoryPricingPolicies) Get(_ context.Context, id, version string) (domain.PricingPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id+"@"+version]
	if !ok {
		return domain.PricingPolicy{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryPricingPolicies) ListByDeployment(_ context.Context, deploymentID string) ([]domain.PricingPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.PricingPolicy, 0)
	for _, item := range r.items {
		if item.DeploymentID == deploymentID {
			items = append(items, item)
		}
	}
	sortPolicies(items)
	return items, nil
}

func (r *MemoryPricingPolicies) Resolve(_ context.Context, deploymentID string, at time.Time) (domain.PricingPolicy, error) {
	items, _ := r.ListByDeployment(context.Background(), deploymentID)
	for _, item := range items {
		if !at.Before(item.EffectiveFrom) && (item.EffectiveTo == nil || at.Before(*item.EffectiveTo)) {
			return item, nil
		}
	}
	return domain.PricingPolicy{}, ErrNotFound
}

func sortPolicies(items []domain.PricingPolicy) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].EffectiveFrom.Equal(items[j].EffectiveFrom) {
			return items[i].Version > items[j].Version
		}
		return items[i].EffectiveFrom.After(items[j].EffectiveFrom)
	})
}
