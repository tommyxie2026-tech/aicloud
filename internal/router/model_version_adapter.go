package router

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type modelVersionAdapter struct {
	repo domain.ModelVersionRepository
}

func (a modelVersionAdapter) List(ctx context.Context) ([]domain.Model, error) {
	items, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	models := make([]domain.Model, 0, len(items))
	for _, item := range items {
		models = append(models, item.LegacyModel(""))
	}
	return models, nil
}

func (a modelVersionAdapter) Get(ctx context.Context, id string) (domain.Model, error) {
	item, err := a.repo.Get(ctx, id)
	if err != nil {
		return domain.Model{}, err
	}
	return item.LegacyModel(""), nil
}

func (a modelVersionAdapter) Create(context.Context, domain.Model) (domain.Model, error) {
	return domain.Model{}, fmt.Errorf("model version routing adapter is read-only")
}

func (a modelVersionAdapter) Update(context.Context, domain.Model) (domain.Model, error) {
	return domain.Model{}, fmt.Errorf("model version routing adapter is read-only")
}
