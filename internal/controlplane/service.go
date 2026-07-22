package controlplane

import (
	"context"
	"fmt"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"time"
)

type Service struct {
	models *modelservice.Service
	tasks  domain.TaskRepository
	engine workflow.Engine
}

func New(models *modelservice.Service, tasks domain.TaskRepository, engine workflow.Engine) *Service {
	return &Service{models: models, tasks: tasks, engine: engine}
}
func (s *Service) ListModels(ctx context.Context) ([]domain.Model, error) { return s.models.List(ctx) }
func (s *Service) CreateModel(ctx context.Context, model domain.Model) (domain.Model, error) {
	return s.models.Create(ctx, model)
}
func (s *Service) ListTasks(ctx context.Context) ([]domain.Task, error) { return s.tasks.List(ctx) }
func (s *Service) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return s.tasks.Get(ctx, id)
}
func (s *Service) CreateTask(ctx context.Context, input, agentID string) (domain.Task, error) {
	now := time.Now().UTC()
	task := domain.Task{ID: fmt.Sprintf("task-%d", now.UnixNano()), AgentID: agentID, Input: input, Status: domain.TaskPending, TraceID: fmt.Sprintf("trace-%d", now.UnixNano()), CreatedAt: now, UpdatedAt: now}
	created, err := s.tasks.Create(ctx, task)
	if err != nil {
		return domain.Task{}, err
	}
	if s.engine != nil {
		_ = s.engine.Start(ctx, created.ID)
	}
	return created, nil
}
