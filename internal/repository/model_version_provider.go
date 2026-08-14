package repository

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func (r *PostgresModels) ModelVersionRepository() domain.ModelVersionRepository {
	if r == nil {
		return nil
	}
	return NewPostgresModelVersions(r.db)
}

func (r *MemoryModels) ModelVersionRepository() domain.ModelVersionRepository {
	if r == nil {
		return nil
	}
	return legacyModelVersions{models: r}
}

type legacyModelVersions struct {
	models domain.ModelRepository
}

func (r legacyModelVersions) List(ctx context.Context) ([]domain.ModelVersion, error) {
	models, err := r.models.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ModelVersion, 0, len(models))
	for _, model := range models {
		items = append(items, domain.ModelVersionFromLegacy(model))
	}
	return items, nil
}

func (r legacyModelVersions) Get(ctx context.Context, id string) (domain.ModelVersion, error) {
	model, err := r.models.Get(ctx, id)
	if err != nil {
		return domain.ModelVersion{}, err
	}
	return domain.ModelVersionFromLegacy(model), nil
}

func (legacyModelVersions) Create(context.Context, domain.ModelVersion) (domain.ModelVersion, error) {
	return domain.ModelVersion{}, fmt.Errorf("legacy model-version projection is read-only")
}

func (legacyModelVersions) Update(context.Context, domain.ModelVersion) (domain.ModelVersion, error) {
	return domain.ModelVersion{}, fmt.Errorf("legacy model-version projection is read-only")
}
