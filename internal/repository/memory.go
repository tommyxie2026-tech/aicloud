package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var (
	ErrNotFound        = errors.New("resource not found")
	ErrVersionConflict = errors.New("resource version conflict")
)

type MemoryModels struct {
	mu               sync.RWMutex
	m                map[string]domain.Model
	licenseEvidence  *MemoryLicenseEvidenceVersions
}

func NewMemoryModels(seed ...domain.Model) *MemoryModels {
	r := &MemoryModels{
		m:               make(map[string]domain.Model, len(seed)),
		licenseEvidence: NewMemoryLicenseEvidenceVersions(),
	}
	for _, model := range seed {
		r.m[model.ID] = model
	}
	return r
}

func (r *MemoryModels) List(_ context.Context) ([]domain.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Model, 0, len(r.m))
	for _, model := range r.m {
		items = append(items, model)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryModels) Get(_ context.Context, id string) (domain.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.m[id]
	if !ok {
		return domain.Model{}, ErrNotFound
	}
	return model, nil
}

func (r *MemoryModels) Create(_ context.Context, model domain.Model) (domain.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[model.ID]; ok {
		return domain.Model{}, errors.New("model already exists")
	}
	r.m[model.ID] = model
	return model, nil
}

func (r *MemoryModels) Update(_ context.Context, model domain.Model) (domain.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[model.ID]; !ok {
		return domain.Model{}, ErrNotFound
	}
	r.m[model.ID] = model
	return model, nil
}

// remainder of repository types intentionally unchanged below this point.
