package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type MemoryDeployments struct {
	mu    sync.RWMutex
	items map[string]domain.Deployment
}

func NewMemoryDeployments(seed ...domain.Deployment) *MemoryDeployments {
	r := &MemoryDeployments{items: make(map[string]domain.Deployment, len(seed))}
	for _, item := range seed {
		normalizeDeploymentIdentity(&item)
		r.items[item.ID] = item
	}
	return r
}

func (r *MemoryDeployments) List(_ context.Context) ([]domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Deployment, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryDeployments) ListByModel(_ context.Context, modelID, version string) ([]domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Deployment, 0)
	for _, item := range r.items {
		if item.ModelID == modelID && item.ModelVersion == version {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryDeployments) ListByModelVersion(_ context.Context, modelVersionID string) ([]domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Deployment, 0)
	for _, item := range r.items {
		if item.ModelVersionID == modelVersionID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryDeployments) Get(_ context.Context, id string) (domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.Deployment{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryDeployments) Create(_ context.Context, item domain.Deployment) (domain.Deployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; ok {
		return domain.Deployment{}, errors.New("deployment already exists")
	}
	normalizeDeploymentIdentity(&item)
	r.items[item.ID] = item
	return item, nil
}

func (r *MemoryDeployments) Update(_ context.Context, item domain.Deployment) (domain.Deployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return domain.Deployment{}, ErrNotFound
	}
	normalizeDeploymentIdentity(&item)
	r.items[item.ID] = item
	return item, nil
}

func normalizeDeploymentIdentity(item *domain.Deployment) {
	if item != nil && item.ModelVersionID == "" {
		item.ModelVersionID = item.ModelID
	}
}
