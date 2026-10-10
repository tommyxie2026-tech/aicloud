package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type MemoryLicenseEvidenceVersions struct {
	mu    sync.RWMutex
	items map[string]domain.LicenseEvidenceVersion
}

func NewMemoryLicenseEvidenceVersions() *MemoryLicenseEvidenceVersions {
	return &MemoryLicenseEvidenceVersions{items: make(map[string]domain.LicenseEvidenceVersion)}
}

func (r *MemoryLicenseEvidenceVersions) Create(_ context.Context, item domain.LicenseEvidenceVersion) (domain.LicenseEvidenceVersion, error) {
	if err := item.Validate(); err != nil {
		return domain.LicenseEvidenceVersion{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[item.Ref()]; exists {
		return domain.LicenseEvidenceVersion{}, errors.New("license evidence version already exists")
	}
	r.items[item.Ref()] = item
	return item, nil
}

func (r *MemoryLicenseEvidenceVersions) Get(_ context.Context, id, version string) (domain.LicenseEvidenceVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id+"@"+version]
	if !ok {
		return domain.LicenseEvidenceVersion{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryLicenseEvidenceVersions) ListByModelVersion(_ context.Context, modelVersionID string) ([]domain.LicenseEvidenceVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.LicenseEvidenceVersion, 0)
	for _, item := range r.items {
		if item.ModelVersionID == modelVersionID {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].EffectiveFrom.Equal(items[j].EffectiveFrom) {
			return items[i].Version > items[j].Version
		}
		return items[i].EffectiveFrom.After(items[j].EffectiveFrom)
	})
	return items, nil
}

func (r *MemoryLicenseEvidenceVersions) Resolve(_ context.Context, modelVersionID string, at time.Time) (domain.LicenseEvidenceVersion, error) {
	items, _ := r.ListByModelVersion(context.Background(), modelVersionID)
	for _, item := range items {
		if !at.Before(item.EffectiveFrom) && (item.EffectiveTo == nil || at.Before(*item.EffectiveTo)) {
			return item, nil
		}
	}
	return domain.LicenseEvidenceVersion{}, ErrNotFound
}
