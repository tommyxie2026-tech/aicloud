package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type MemoryModelVersions struct {
	mu    sync.RWMutex
	items map[string]domain.ModelVersion
}

func NewMemoryModelVersions(seed ...domain.ModelVersion) *MemoryModelVersions {
	r := &MemoryModelVersions{items: make(map[string]domain.ModelVersion, len(seed))}
	for _, item := range seed {
		r.items[item.ID] = item
	}
	return r
}

func (r *MemoryModelVersions) List(_ context.Context) ([]domain.ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.ModelVersion, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryModelVersions) Get(_ context.Context, id string) (domain.ModelVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return domain.ModelVersion{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryModelVersions) Create(_ context.Context, item domain.ModelVersion) (domain.ModelVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; ok {
		return domain.ModelVersion{}, errors.New("model version already exists")
	}
	r.items[item.ID] = item
	return item, nil
}

func (r *MemoryModelVersions) Update(_ context.Context, item domain.ModelVersion) (domain.ModelVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[item.ID]; !ok {
		return domain.ModelVersion{}, ErrNotFound
	}
	r.items[item.ID] = item
	return item, nil
}
