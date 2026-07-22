package modelservice

import (
	"context"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"time"
)

type Service struct{ repo domain.ModelRepository }

func New(repo domain.ModelRepository) *Service                      { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context) ([]domain.Model, error) { return s.repo.List(ctx) }
func (s *Service) Create(ctx context.Context, model domain.Model) (domain.Model, error) {
	if model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now().UTC()
	}
	return s.repo.Create(ctx, model)
}
