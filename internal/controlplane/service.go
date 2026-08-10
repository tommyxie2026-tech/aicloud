package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/admission"
	"github.com/tommyxie2026-tech/aicloud/internal/audit"
	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/evaluation"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/modelruntime"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
	"github.com/tommyxie2026-tech/aicloud/internal/toolgateway"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

type Service struct {
	models       *modelservice.Service
	tasks        domain.TaskRepository
	engine       workflow.Engine
	router       *router.Router
	routes       domain.RouteDecisionRepository
	costEvents   domain.CostEventRepository
	costLedger   *cost.Ledger
	tools        *toolgateway.Service
	audit        audit.Store
	traces       tracepkg.Store
	evaluations  evaluation.Store
	admission    *admission.Service
	modelRuntime *modelruntime.Executor
}

func New(models *modelservice.Service, tasks domain.TaskRepository, engine workflow.Engine) *Service {
	return &Service{models: models, tasks: tasks, engine: engine}
}

func (s *Service) WithGovernance(routes domain.RouteDecisionRepository, costs domain.CostEventRepository) *Service {
	s.routes = routes
	s.costEvents = costs
	if costs != nil {
		s.costLedger = cost.New(costs)
	}
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

func (s *Service) WithEvidence(traceStore tracepkg.Store, evaluationStore evaluation.Store, admissionService *admission.Service, runtime *modelruntime.Executor) *Service {
	s.traces = traceStore
	s.evaluations = evaluationStore
	s.admission = admissionService
	s.modelRuntime = runtime
	if s.router != nil && admissionService != nil {
		s.router.WithAdmission(admissionService)
	}
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

func (s *Service) AppendAdmissionEvidence(ctx context.Context, modelID string, evidence admission.Evidence) (admission.Evidence, error) {
	if s.admission == nil {
		return admission.Evidence{}, fmt.Errorf("model admission is not configured")
	}
	model, err := s.models.Get(ctx, modelID)
	if err != nil {
		return admission.Evidence{}, err
	}
	evidence.ModelID = model.ID
	evidence.ModelVersion = model.Version
	if evidence.CreatedAt.IsZero() {
		evidence.CreatedAt = time.Now().UTC()
	}
	if err := s.admission.Append(ctx, evidence); err != nil {
		return admission.Evidence{}, err
	}
	return evidence, nil
}

func (s *Service) ListAdmissionEvidence(ctx context.Context, modelID string) ([]admission.Evidence, error) {
	if s.admission == nil {
		return nil, fmt.Errorf("model admission is not configured")
	}
	model, err := s.models.Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	return s.admission.List(ctx, model.ID, model.Version)
}

func (s *Service) CheckModelAdmission(ctx context.Context, modelID string) (admission.Decision, error) {
	if s.admission == nil {
		return admission.Decision{}, fmt.Errorf("model admission is not configured")
	}
	model, err := s.models.Get(ctx, modelID)
	if err != nil {
		return admission.Decision{}, err
	}
	return s.admission.Check(ctx, model)
}

func (s *Service) ListTasks(ctx context.Context) ([]domain.Task, error) {
	return s.tasks.List(ctx)
}

func (s *Service) GetTask(ctx context.Context, id string) (domain.Task, error) {
	return s.tasks.Get(ctx, id)
}

func (s *Service) CreateTask(ctx context.Context, input, agentID string) (domain.Task, error) {
	now := time.Now().UTC()
	task, err := domain.NewTask(domain.NewTaskParams{
		ID:       fmt.Sprintf("task-%d", now.UnixNano()),
		AgentID:  agentID,
		Input:    input,
		TraceID:  fmt.Sprintf("trace-%d", now.UnixNano()),
		Currency: "USD",
		Now:      now,
	})
	if err != nil {
		return domain.Task{}, err
	}
	created, err := s.tasks.Create(ctx, task)
	if err != nil {
		return domain.Task{}, err
	}
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: created.TraceID, TaskID: created.ID,
		SpanID: tracepkg.NewID("span"), Name: "task.created", Kind: "TASK",
		Status: tracepkg.StatusOK, Attributes: map[string]string{
			"agent.id": agentID, "task.status": string(created.Status),
			"task.version": fmt.Sprintf("%d", created.Version),
		}, StartedAt: now, EndedAt: timePointer(now),
	})
	if s.engine != nil {
		_ = s.engine.Start(ctx, created.ID)
	}
	return created, nil
}

func (s *Service) DecideRoute(ctx context.Context, request router.Request) (domain.RouteDecision, error) {
	if s.router == nil {
		return domain.RouteDecision{}, fmt.Errorf("routing is not configured")
	}
	task, err := s.tasks.Get(ctx, request.TaskID)
	if err != nil {
		return domain.RouteDecision{}, err
	}

	switch task.Status {
	case domain.TaskCreated:
		task, err = s.transitionTask(ctx, task, domain.TaskPlanning, "prepare task plan")
		if err == nil {
			task, err = s.transitionTask(ctx, task, domain.TaskRouting, "request model route")
		}
	case domain.TaskPlanning:
		task, err = s.transitionTask(ctx, task, domain.TaskRouting, "request model route")
	case domain.TaskRouting:
		// A routing retry may reuse the current ROUTING state.
	default:
		err = fmt.Errorf("%w: cannot route task from %s", domain.ErrInvalidTaskTransition, task.Status)
	}
	if err != nil {
		return domain.RouteDecision{}, err
	}

	decision, err := s.router.Decide(ctx, request)
	if err != nil {
		s.appendTrace(ctx, tracepkg.Event{
			ID: tracepkg.NewID("trace-event"), TraceID: task.TraceID, TaskID: task.ID,
			SpanID: tracepkg.NewID("span"), Name: "route.decision", Kind: "ROUTING",
			Status: tracepkg.StatusError, Message: err.Error(), StartedAt: time.Now().UTC(),
		})
		return domain.RouteDecision{}, err
	}
	task.RouteDecisionID = decision.ID
	task.EstimatedCost = decision.Selected.EstimatedCost
	task.UpdatedAt = time.Now().UTC()
	task, err = s.tasks.Update(ctx, task)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	now := time.Now().UTC()
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: task.TraceID, TaskID: task.ID,
		SpanID: tracepkg.NewID("span"), Name: "route.decision", Kind: "ROUTING",
		Status: tracepkg.StatusOK, Attributes: map[string]string{
			"route.id": decision.ID, "route.class": string(decision.Selected.RouteClass),
			"model.id": decision.Selected.ModelID, "model.version": decision.Selected.ModelVersion,
			"evidence.version": decision.EvidenceVersion, "policy.version": decision.PolicyVersion,
			"task.version": fmt.Sprintf("%d", task.Version),
		}, StartedAt: now, EndedAt: timePointer(now),
	})
	return decision, nil
}

func (s *Service) ExecuteModel(ctx context.Context, taskID string, request provider.ProviderRequest) (modelruntime.Result, error) {
	if s.modelRuntime == nil {
		return modelruntime.Result{}, fmt.Errorf("model runtime is not configured")
	}
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return modelruntime.Result{}, err
	}
	if task.RouteDecisionID == "" {
		return modelruntime.Result{}, fmt.Errorf("task does not have a route decision")
	}
	decision, err := s.routes.Get(ctx, task.RouteDecisionID)
	if err != nil {
		return modelruntime.Result{}, err
	}
	if request.RequestID == "" {
		request.RequestID = task.ID
	}

	switch task.Status {
	case domain.TaskRouting:
		task, err = s.transitionTask(ctx, task, domain.TaskExecuting, "execute selected model deployment")
	case domain.TaskExecuting:
		// Execution retries may resume an already executing task. Durable retry
		// semantics move to R6/S3; R5 only prevents invalid lifecycle jumps.
	default:
		err = fmt.Errorf("%w: cannot execute model from %s", domain.ErrInvalidTaskTransition, task.Status)
	}
	if err != nil {
		return modelruntime.Result{}, err
	}

	result, executeErr := s.modelRuntime.Execute(ctx, task.ID, task.TraceID, decision, request)
	if executeErr != nil {
		task.Result = executeErr.Error()
	} else {
		task.Result = encodeProviderResult(result.Response)
	}
	if s.costLedger != nil {
		if total, currency, aggregateErr := s.costLedger.AggregateTask(ctx, task.ID); aggregateErr == nil {
			task.ActualCost = total
			task.Cost = total
			if currency != "" {
				task.Currency = currency
			}
		}
	}

	if executeErr != nil {
		if _, transitionErr := s.transitionTask(ctx, task, domain.TaskFailed, "model execution failed"); transitionErr != nil {
			return result, transitionErr
		}
		return result, executeErr
	}

	task, err = s.transitionTask(ctx, task, domain.TaskValidating, "validate model execution result")
	if err != nil {
		return result, err
	}
	if _, err = s.transitionTask(ctx, task, domain.TaskCompleted, "model execution result accepted"); err != nil {
		return result, err
	}
	return result, nil
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
	started := time.Now().UTC()
	result, executeErr := s.tools.Execute(ctx, request)
	ended := time.Now().UTC()
	status := tracepkg.StatusOK
	message := ""
	if executeErr != nil {
		status = tracepkg.StatusError
		message = executeErr.Error()
	}
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: task.TraceID, TaskID: task.ID,
		SpanID: tracepkg.NewID("span"), Name: "tool.execute", Kind: "TOOL_CALL",
		Status: status, Message: message, Attributes: map[string]string{
			"tool.id": request.ToolID, "tool.action": request.Action,
			"sandbox.status": result.SandboxResult.Status,
		}, StartedAt: started, EndedAt: &ended,
	})
	return result, executeErr
}

func (s *Service) ListToolAudit(ctx context.Context, taskID string) ([]audit.Event, error) {
	if s.audit == nil {
		return nil, fmt.Errorf("audit store is not configured")
	}
	return s.audit.ListByTask(ctx, taskID)
}

func (s *Service) CreateEvaluation(ctx context.Context, taskID string, config evaluation.Config, metrics evaluation.Metrics, thresholds evaluation.Thresholds, rawOutput []byte) (evaluation.Run, error) {
	if s.evaluations == nil {
		return evaluation.Run{}, fmt.Errorf("evaluation store is not configured")
	}
	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return evaluation.Run{}, err
	}
	run, err := evaluation.NewRun(task.ID, task.TraceID, config, metrics, thresholds, rawOutput, time.Now().UTC())
	if err != nil {
		return evaluation.Run{}, err
	}
	if err := s.evaluations.Append(ctx, run); err != nil {
		return evaluation.Run{}, err
	}
	now := time.Now().UTC()
	status := tracepkg.StatusOK
	if !run.Gate.Passed {
		status = tracepkg.StatusError
	}
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: task.TraceID, TaskID: task.ID,
		SpanID: tracepkg.NewID("span"), Name: "evaluation.run", Kind: "EVALUATION",
		Status: status, Attributes: map[string]string{
			"evaluation.id": run.ID, "evaluation.config_digest": run.ConfigDigest,
			"evaluation.gate_passed": fmt.Sprintf("%t", run.Gate.Passed),
			"dataset.version":        run.Config.DatasetVersion, "evaluator.version": run.Config.EvaluatorVersion,
		}, StartedAt: now, EndedAt: timePointer(now),
	})
	return run, nil
}

func (s *Service) ListEvaluations(ctx context.Context, taskID string) ([]evaluation.Run, error) {
	if s.evaluations == nil {
		return nil, fmt.Errorf("evaluation store is not configured")
	}
	return s.evaluations.ListByTask(ctx, taskID)
}

func (s *Service) ListTrace(ctx context.Context, taskID string) ([]tracepkg.Event, error) {
	if s.traces == nil {
		return nil, fmt.Errorf("trace store is not configured")
	}
	if _, err := s.tasks.Get(ctx, taskID); err != nil {
		return nil, err
	}
	return s.traces.ListByTask(ctx, taskID)
}

func (s *Service) transitionTask(ctx context.Context, task domain.Task, target domain.TaskStatus, cause string) (domain.Task, error) {
	now := time.Now().UTC()
	transition, err := task.Transition(domain.TaskTransitionCommand{
		To: target, Actor: transitionActor(ctx), Cause: cause, At: now,
	})
	if err != nil {
		return domain.Task{}, err
	}
	updated, err := s.tasks.Update(ctx, task)
	if err != nil {
		return domain.Task{}, err
	}
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: updated.TraceID, TaskID: updated.ID,
		SpanID: tracepkg.NewID("span"), Name: "task.transition", Kind: "TASK",
		Status: tracepkg.StatusOK, Attributes: map[string]string{
			"task.from": string(transition.From), "task.to": string(transition.To),
			"task.actor": transition.Actor, "task.cause": transition.Cause,
			"task.version": fmt.Sprintf("%d", updated.Version),
		}, StartedAt: now, EndedAt: timePointer(now),
	})
	return updated, nil
}

func transitionActor(ctx context.Context) string {
	if principal, ok := identity.PrincipalFromContext(ctx); ok {
		return fmt.Sprintf("%s:%s", principal.Type, principal.SubjectID)
	}
	// This fallback is evidence metadata only; it does not grant authorization.
	// Scoped repositories still require an explicit Principal for protected data.
	return "controlplane"
}

func (s *Service) appendTrace(ctx context.Context, event tracepkg.Event) {
	if s.traces != nil {
		_ = s.traces.Append(ctx, event)
	}
}

func encodeProviderResult(response *provider.ProviderResponse) string {
	if response == nil {
		return ""
	}
	body, err := json.Marshal(response)
	if err != nil {
		return response.RawText
	}
	return string(body)
}

func timePointer(value time.Time) *time.Time { return &value }
