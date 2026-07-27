package modelservice

import (
	"context"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type Service struct{ repo domain.ModelRepository }

func New(repo domain.ModelRepository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]domain.Model, error) { return s.repo.List(ctx) }

func (s *Service) Get(ctx context.Context, id string) (domain.Model, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, model domain.Model) (domain.Model, error) {
	now := time.Now().UTC()
	applyDefaults(&model, now)
	return s.repo.Create(ctx, model)
}

func (s *Service) Update(ctx context.Context, model domain.Model) (domain.Model, error) {
	current, err := s.repo.Get(ctx, model.ID)
	if err != nil {
		return domain.Model{}, err
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = current.CreatedAt
	}
	applyDefaults(&model, time.Now().UTC())
	return s.repo.Update(ctx, model)
}

func applyDefaults(model *domain.Model, now time.Time) {
	if model.Version == "" {
		model.Version = "v1"
	}
	if model.Lifecycle == "" {
		model.Lifecycle = domain.ModelDraft
	}
	if model.Health == "" {
		model.Health = domain.HealthUnknown
	}
	if model.ApprovalStatus == "" {
		model.ApprovalStatus = domain.ApprovalDiscovered
	}
	if model.Pricing.Currency == "" {
		model.Pricing.Currency = "USD"
	}
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	model.UpdatedAt = now
}
