package controlplane

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
)

type serviceModelVersions struct {
	models *modelservice.Service
}

func (a serviceModelVersions) List(ctx context.Context) ([]domain.ModelVersion, error) {
	models, err := a.models.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ModelVersion, 0, len(models))
	for _, model := range models {
		items = append(items, domain.ModelVersionFromLegacy(model))
	}
	return items, nil
}

func (a serviceModelVersions) Get(ctx context.Context, id string) (domain.ModelVersion, error) {
	model, err := a.models.Get(ctx, id)
	if err != nil {
		return domain.ModelVersion{}, err
	}
	return domain.ModelVersionFromLegacy(model), nil
}

func (serviceModelVersions) Create(context.Context, domain.ModelVersion) (domain.ModelVersion, error) {
	return domain.ModelVersion{}, fmt.Errorf("control-plane model-version projection is read-only")
}

func (serviceModelVersions) Update(context.Context, domain.ModelVersion) (domain.ModelVersion, error) {
	return domain.ModelVersion{}, fmt.Errorf("control-plane model-version projection is read-only")
}
