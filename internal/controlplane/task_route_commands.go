package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
)

type RouteCommandResult struct {
	Decision domain.RouteDecision
	Replayed bool
}

// DecideRouteIdempotent is the R6 public routing path. In durable PostgreSQL
// mode it separates volatile route planning from persistence, then commits the
// RouteDecision and PLANNING->ROUTING Task mutation in one transaction with the
// canonical TaskEvent and command Idempotency result.
func (s *Service) DecideRouteIdempotent(ctx context.Context, request router.Request, metadata CommandMetadata) (RouteCommandResult, error) {
	if err := metadata.Validate(); err != nil {
		return RouteCommandResult{}, err
	}
	if s.router == nil {
		return RouteCommandResult{}, fmt.Errorf("routing is not configured")
	}
	commands := s.taskCommandStore()
	if commands == nil {
		decision, err := s.DecideRoute(ctx, request)
		return RouteCommandResult{Decision: decision}, err
	}

	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return RouteCommandResult{}, err
	}
	operation := routeCommandOperation(request.TaskID)
	if replay, found, err := resolveRouteReplay(ctx, commands, principal, operation, metadata); err != nil {
		return RouteCommandResult{}, err
	} else if found {
		return replay, nil
	}

	task, err := s.tasks.Get(ctx, request.TaskID)
	if err != nil {
		return RouteCommandResult{}, err
	}

	switch task.Status {
	case domain.TaskCreated:
		task, err = s.commitPlanningForRoute(ctx, commands, principal, task, metadata)
		if err != nil {
			return RouteCommandResult{}, err
		}
	case domain.TaskPlanning:
		// Already prepared by an earlier attempt; continue the same logical route command.
	case domain.TaskRouting:
		// A concurrent duplicate may have committed after the initial replay lookup.
		if replay, found, replayErr := resolveRouteReplay(ctx, commands, principal, operation, metadata); replayErr != nil {
			return RouteCommandResult{}, replayErr
		} else if found {
			return replay, nil
		}
		return RouteCommandResult{}, fmt.Errorf("%w: routing state has no matching durable command result", domain.ErrInvalidTaskTransition)
	default:
		return RouteCommandResult{}, fmt.Errorf("%w: cannot route task from %s", domain.ErrInvalidTaskTransition, task.Status)
	}

	decision, err := s.router.Plan(ctx, request)
	if err != nil {
		return RouteCommandResult{}, err
	}

	now := time.Now().UTC()
	transition, err := task.Transition(domain.TaskTransitionCommand{
		To: domain.TaskRouting, Actor: transitionActor(ctx), Cause: "request model route", At: now,
	})
	if err != nil {
		return RouteCommandResult{}, err
	}
	task.RouteDecisionID = decision.ID
	task.EstimatedCost = decision.Selected.EstimatedCost

	eventPayload, err := json.Marshal(map[string]any{
		"from":                  transition.From,
		"to":                    transition.To,
		"routeDecisionId":       decision.ID,
		"selectedModelId":       decision.Selected.ModelID,
		"selectedModelVersion":  decision.Selected.ModelVersion,
		"selectedDeploymentId":  decision.Selected.DeploymentID,
		"estimatedInputTokens":  request.EstimatedInputTokens,
		"estimatedOutputTokens": request.EstimatedOutputTokens,
		"serviceTier":           request.ServiceTier,
		"inferenceEffort":       request.InferenceEffort,
	})
	if err != nil {
		return RouteCommandResult{}, fmt.Errorf("encode route TaskEvent payload: %w", err)
	}

	commit := repository.RouteTaskCommandCommit{
		Task:       task,
		Transition: transition,
		Decision:   decision,
		Event: domain.TaskEvent{
			EventID:       tracepkg.NewID("task-event"),
			EventType:     "TaskRoutingStarted",
			Actor:         principalTaskEventActor(principal),
			Payload:       eventPayload,
			RequestID:     strings.TrimSpace(metadata.RequestID),
			SchemaVersion: 1,
			OccurredAt:    now,
			CreatedAt:     now,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID:      principal.TenantID,
			ProjectID:     principal.ProjectID,
			Operation:     operation,
			Key:           strings.TrimSpace(metadata.IdempotencyKey),
			RequestDigest: strings.TrimSpace(metadata.RequestDigest),
			Status:        domain.IdempotencyCompleted,
			ResponseCode:  201,
			CreatedAt:     now,
			ExpiresAt:     now.Add(24 * time.Hour),
		},
	}
	result, err := commands.CommitRouteTransition(ctx, commit)
	if err != nil {
		return RouteCommandResult{}, err
	}
	if !result.Replayed {
		s.appendTrace(ctx, tracepkg.Event{
			ID: tracepkg.NewID("trace-event"), TraceID: result.Task.TraceID, TaskID: result.Task.ID,
			SpanID: tracepkg.NewID("span"), Name: "route.decision", Kind: "ROUTING",
			Status: tracepkg.StatusOK, Attributes: map[string]string{
				"route.id":           result.Decision.ID,
				"route.class":        string(result.Decision.Selected.RouteClass),
				"model.id":           result.Decision.Selected.ModelID,
				"model.version":      result.Decision.Selected.ModelVersion,
				"deployment.id":      result.Decision.Selected.DeploymentID,
				"evidence.version":   result.Decision.EvidenceVersion,
				"policy.version":     result.Decision.PolicyVersion,
				"task.version":       fmt.Sprintf("%d", result.Task.Version),
				"command.idempotent": "true",
			}, StartedAt: now, EndedAt: timePointer(now),
		})
	}
	return RouteCommandResult{Decision: result.Decision, Replayed: result.Replayed}, nil
}

func (s *Service) commitPlanningForRoute(ctx context.Context, commands repository.TaskCommandStore, principal identity.Principal, task domain.Task, metadata CommandMetadata) (domain.Task, error) {
	now := time.Now().UTC()
	transition, err := task.Transition(domain.TaskTransitionCommand{
		To: domain.TaskPlanning, Actor: transitionActor(ctx), Cause: "prepare task plan", At: now,
	})
	if err != nil {
		return domain.Task{}, err
	}
	payload, err := json.Marshal(map[string]any{
		"from":  transition.From,
		"to":    transition.To,
		"cause": transition.Cause,
	})
	if err != nil {
		return domain.Task{}, fmt.Errorf("encode planning TaskEvent payload: %w", err)
	}
	commit := repository.TaskCommandCommit{
		Task:       task,
		Transition: transition,
		Event: domain.TaskEvent{
			EventID:       tracepkg.NewID("task-event"),
			EventType:     "TaskPlanningStarted",
			Actor:         principalTaskEventActor(principal),
			Payload:       payload,
			RequestID:     strings.TrimSpace(metadata.RequestID),
			SchemaVersion: 1,
			OccurredAt:    now,
			CreatedAt:     now,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID:      principal.TenantID,
			ProjectID:     principal.ProjectID,
			Operation:     "task:route:planning:" + task.ID,
			Key:           strings.TrimSpace(metadata.IdempotencyKey) + ":planning",
			RequestDigest: strings.TrimSpace(metadata.RequestDigest),
			Status:        domain.IdempotencyCompleted,
			ResourceID:    task.ID,
			ResponseCode:  200,
			CreatedAt:     now,
			ExpiresAt:     now.Add(24 * time.Hour),
		},
	}
	result, err := commands.CommitTransition(ctx, commit)
	if err != nil {
		return domain.Task{}, err
	}
	if result.Replayed {
		return s.tasks.Get(ctx, task.ID)
	}
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: result.Task.TraceID, TaskID: result.Task.ID,
		SpanID: tracepkg.NewID("span"), Name: "task.transition", Kind: "TASK",
		Status: tracepkg.StatusOK, Attributes: map[string]string{
			"task.from": string(transition.From), "task.to": string(transition.To),
			"task.actor": transition.Actor, "task.cause": transition.Cause,
			"task.version": fmt.Sprintf("%d", result.Task.Version),
		}, StartedAt: now, EndedAt: timePointer(now),
	})
	return result.Task, nil
}

func resolveRouteReplay(ctx context.Context, commands repository.TaskCommandStore, principal identity.Principal, operation string, metadata CommandMetadata) (RouteCommandResult, bool, error) {
	record, found, err := commands.ResolveIdempotency(ctx, repository.IdempotencyLookup{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		Operation: operation, Key: strings.TrimSpace(metadata.IdempotencyKey),
		RequestDigest: strings.TrimSpace(metadata.RequestDigest),
	})
	if err != nil || !found {
		return RouteCommandResult{}, false, err
	}
	if record.Status != domain.IdempotencyCompleted || len(record.ResponsePayload) == 0 {
		return RouteCommandResult{}, false, fmt.Errorf("route replay does not contain a completed response")
	}
	var decision domain.RouteDecision
	if err := json.Unmarshal(record.ResponsePayload, &decision); err != nil {
		return RouteCommandResult{}, false, fmt.Errorf("decode route replay response: %w", err)
	}
	return RouteCommandResult{Decision: decision, Replayed: true}, true, nil
}

func routeCommandOperation(taskID string) string {
	return "POST /api/v1/tasks/" + strings.TrimSpace(taskID) + "/route"
}

func principalTaskEventActor(principal identity.Principal) domain.TaskEventActor {
	return domain.TaskEventActor{PrincipalType: string(principal.Type), SubjectID: principal.SubjectID}
}
