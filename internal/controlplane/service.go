package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
	"github.com/tommyxie2026-tech/aicloud/internal/toolgateway"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

type Service struct {
	models     *modelservice.Service
	tasks      domain.TaskRepository
	engine     workflow.Engine
	router     *router.Router
	routes     domain.RouteDecisionRepository
	costEvents domain.CostEventRepository
	tools      *toolgateway.Service
	audit      audit.Store
}

func New(models *modelservice.Service, tasks domain.TaskRepository, engine workflow.Engine) *Service {
	return &Service{models: models, tasks: tasks, engine: engine}
}

func (s *Service) WithGovernance(routes domain.RouteDecisionRepository, costs domain.CostEventRepository) *Service {
	s.routes = routes
	s.costEvents = costs
	if routes != nil {
		s.router = router.New(s.modelsRepository(), routes)
	}
	return s
}

func (s *Service) WithSecureTools(tools *toolgateway.Service, auditStore audit.Store) *Service {
	s.tools = tools
	s.audit = auditStore
	return s
}

func (s *Service) modelsRepository() domain.ModelRepository {
	return modelRepositoryAdapter{service: s.models}
}

type modelRepositoryAdapter struct{ service *modelservice.Service }

func (a modelRepositoryAdapter) List(ctx context.Context) ([]domain.Model, error) {
	return a.service.List(ctx)
}
func (a modelRepositoryAdapter) Get(ctx context.Context, id string) (domain.Model, error) {
	return a.service.Get(ctx, id)
}
func (a modelRepositoryAdapter) Create(ctx context.Context, model domain.Model) (domain.Model, error) {
	return a.service.Create(ctx, model)
}
func (a modelRepositoryAdapter) Update(ctx context.Context, model domain.Model) (domain.Model, error) {
	return a.service.Update(ctx, model)
}

func (s *Service) ListModels(ctx context.Context) ([]domain.Model, error) {
	return s.models.List(ctx)
}

func (s *Service) GetModel(ctx context.Context, id string) (domain.Model, error) {
	return s.models.Get(ctx, id)
}

func (s *Service) CreateModel(ctx context.Context, model domain.Model) (domain.Model, error) {
	return s.models.Create(ctx, model)
}

func (s *Service) UpdateModel(ctx context.Context, model domain.Model) (domain.Model, error) {
	return s.models.Update(ctx, model)
}

func (s *Service) ListTasks(ctx context.Context) ([]domain.Task, error) {
	return s.tasks.List(ctx)
}

func (s *Service) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return s.tasks.Get(ctx, id)
}

func (s *Service) CreateTask(ctx context.Context, input, agentID string) (domain.Task, error) {
	now := time.Now().UTC()
	task := domain.Task{
		ID:        fmt.Sprintf("task-%d", now.UnixNano()),
		AgentID:   agentID,
		Input:     input,
		Status:    domain.TaskPending,
		Currency:  "USD",
		TraceID:   fmt.Sprintf("trace-%d", now.UnixNano()),
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := s.tasks.Create(ctx, task)
	if err != nil {
		return domain.Task{}, err
	}
	if s.engine != nil {
		_ = s.engine.Start(ctx, created.ID)
	}
	return created, nil
}

func (s *Service) DecideRoute(ctx context.Context, request router.Request) (domain.RouteDecision, error) {
	if s.router == nil {
		return domain.RouteDecision{}, fmt.Errorf("routing is not configured")
	}
	if _, err := s.tasks.Get(ctx, request.TaskID); err != nil {
		return domain.RouteDecision{}, err
	}
	decision, err := s.router.Decide(ctx, request)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	task, err := s.tasks.Get(ctx, request.TaskID)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	task.RouteDecisionID = decision.ID
	task.EstimatedCost = decision.Selected.EstimatedCost
	task.UpdatedAt = time.Now().UTC()
	if _, err := s.tasks.Update(ctx, task); err != nil {
		return domain.RouteDecision{}, err
	}
	return decision, nil
}

func (s *Service) ListRouteDecisions(ctx context.Context, taskID string) ([]domain.RouteDecision, error) {
	if s.routes == nil {
		return nil, fmt.Errorf("route decision repository is not configured")
	}
	return s.routes.ListByTask(ctx, taskID)
}

func (s *Service) ListCostEvents(ctx context.Context, taskID string) ([]domain.CostEvent, error) {
	if s.costEvents == nil {
		return nil, fmt.Errorf("cost event repository is not configured")
	}
	return s.costEvents.ListByTask(ctx, taskID)
}

func (s *Service) ListTools(ctx context.Context) ([]toolgateway.Definition, error) {
	if s.tools == nil {
		return nil, fmt.Errorf("Tool Gateway is not configured")
	}
	return s.tools.List(ctx)
}

func (s *Service) ExecuteTool(ctx context.Context, taskID string, request toolgateway.Request) (toolgateway.Result, error) {
	if s.tools == nil {
		return toolgateway.Result{}, fmt.Errorf("Tool Gateway is not configured")
	}
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return toolgateway.Result{}, err
	}
	request.TaskID = task.ID
	request.TraceID = task.TraceID
	if request.AgentID == "" {
		request.AgentID = task.AgentID
	}
	return s.tools.Execute(ctx, request)
}

func (s *Service) ListToolAudit(ctx context.Context, taskID string) ([]audit.Event, error) {
	if s.audit == nil {
		return nil, fmt.Errorf("audit store is not configured")
	}
	return s.audit.ListByTask(ctx, taskID)
}
