package modelruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/circuitbreaker"
	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

type ProviderRegistry interface {
	Get(context.Context, string) (provider.ModelProvider, error)
	Put(context.Context, string, provider.ModelProvider) error
}

type MemoryProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]provider.ModelProvider
}

func NewMemoryProviderRegistry() *MemoryProviderRegistry {
	return &MemoryProviderRegistry{providers: make(map[string]provider.ModelProvider)}
}

func (r *MemoryProviderRegistry) Get(_ context.Context, modelID string) (provider.ModelProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.providers[modelID]
	if !ok {
		return nil, fmt.Errorf("provider for model %s is not registered", modelID)
	}
	return item, nil
}

func (r *MemoryProviderRegistry) Put(_ context.Context, modelID string, item provider.ModelProvider) error {
	if modelID == "" || item == nil {
		return fmt.Errorf("model ID and provider are required")
	}
	r.mu.Lock()
	r.providers[modelID] = item
	r.mu.Unlock()
	return nil
}

type Attempt struct {
	OperationID  string                     `json:"operationId"`
	AttemptID    string                     `json:"attemptId"`
	ModelID      string                     `json:"modelId"`
	ModelVersion string                     `json:"modelVersion"`
	Status       string                     `json:"status"`
	ErrorCode    provider.ProviderErrorCode `json:"errorCode,omitempty"`
	ErrorMessage string                     `json:"errorMessage,omitempty"`
	Retryable    bool                       `json:"retryable"`
	CircuitState circuitbreaker.State       `json:"circuitState"`
	LatencyMS    int64                      `json:"latencyMs"`
	StartedAt    time.Time                  `json:"startedAt"`
	CompletedAt  time.Time                  `json:"completedAt"`
}

type Result struct {
	Candidate domain.RouteCandidate      `json:"candidate"`
	Response  *provider.ProviderResponse `json:"response,omitempty"`
	Attempts  []Attempt                  `json:"attempts"`
	Fallback  bool                       `json:"fallback"`
}

type Executor struct {
	providers ProviderRegistry
	models    domain.ModelRepository
	breaker   *circuitbreaker.Breaker
	costs     *cost.Ledger
	traces    tracepkg.Store
	now       func() time.Time
}

func NewExecutor(providers ProviderRegistry, models domain.ModelRepository, breaker *circuitbreaker.Breaker, costs *cost.Ledger, traces tracepkg.Store) *Executor {
	return &Executor{providers: providers, models: models, breaker: breaker, costs: costs, traces: traces, now: time.Now}
}

func (e *Executor) Execute(ctx context.Context, taskID, traceID string, decision domain.RouteDecision, request provider.ProviderRequest) (Result, error) {
	if taskID == "" || traceID == "" {
		return Result{}, fmt.Errorf("task ID and trace ID are required")
	}
	if decision.Selected.RouteClass == domain.RouteDeterministic {
		return Result{Candidate: decision.Selected}, nil
	}
	candidates := append([]domain.RouteCandidate{decision.Selected}, decision.FallbackChain...)
	attempts := make([]Attempt, 0, len(candidates))
	var lastErr error
	for index, candidate := range candidates {
		attempt, response, retryable, err := e.executeCandidate(ctx, taskID, traceID, candidate, request, index+1)
		attempts = append(attempts, attempt)
		if err == nil {
			return Result{Candidate: candidate, Response: response, Attempts: attempts, Fallback: index > 0}, nil
		}
		lastErr = err
		if !retryable {
			return Result{Candidate: candidate, Attempts: attempts, Fallback: index > 0}, err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no executable route candidate")
	}
	return Result{Attempts: attempts, Fallback: len(attempts) > 1}, lastErr
}

func (e *Executor) executeCandidate(ctx context.Context, taskID, traceID string, candidate domain.RouteCandidate, request provider.ProviderRequest, attemptNumber int) (Attempt, *provider.ProviderResponse, bool, error) {
	started := e.now().UTC()
	operationID := request.RequestID
	if operationID == "" {
		operationID = taskID
	}
	attempt := Attempt{
		OperationID: operationID,
		AttemptID:   fmt.Sprintf("%s:attempt:%d", operationID, attemptNumber),
		ModelID:     candidate.ModelID, ModelVersion: candidate.ModelVersion,
		Status: "STARTED", StartedAt: started,
	}
	key := candidate.ModelID + "@" + candidate.ModelVersion
	allowed, snapshot, err := e.breaker.Allow(ctx, key)
	if err != nil {
		return attempt, nil, false, err
	}
	attempt.CircuitState = snapshot.State
	if !allowed {
		attempt.Status = "SKIPPED_OPEN_CIRCUIT"
		attempt.Retryable = true
		attempt.CompletedAt = e.now().UTC()
		e.appendTrace(ctx, taskID, traceID, candidate, attempt, "circuit breaker is open")
		return attempt, nil, true, fmt.Errorf("circuit breaker is open for %s", key)
	}
	adapter, err := e.providers.Get(ctx, candidate.ModelID)
	if err != nil {
		attempt.Status = "FAILED"
		attempt.ErrorMessage = err.Error()
		attempt.CompletedAt = e.now().UTC()
		e.appendTrace(ctx, taskID, traceID, candidate, attempt, err.Error())
		return attempt, nil, false, err
	}
	response, err := adapter.Generate(ctx, request)
	completed := e.now().UTC()
	attempt.CompletedAt = completed
	attempt.LatencyMS = completed.Sub(started).Milliseconds()
	if err != nil {
		code, retryable := classify(err)
		attempt.Status = "FAILED"
		attempt.ErrorCode = code
		attempt.ErrorMessage = err.Error()
		attempt.Retryable = retryable
		if retryable {
			snapshot, breakerErr := e.breaker.Failure(ctx, key)
			if breakerErr != nil {
				return attempt, nil, false, breakerErr
			}
			attempt.CircuitState = snapshot.State
		}
		e.appendTrace(ctx, taskID, traceID, candidate, attempt, err.Error())
		return attempt, nil, retryable, err
	}
	if err := e.breaker.Success(ctx, key); err != nil {
		return attempt, nil, false, err
	}
	attempt.Status = "SUCCEEDED"
	attempt.CircuitState = circuitbreaker.StateClosed
	if e.costs != nil {
		model, modelErr := e.models.Get(ctx, candidate.ModelID)
		if modelErr != nil {
			return attempt, nil, false, modelErr
		}
		_, costErr := e.costs.RecordModelUsage(ctx, cost.ModelUsage{
			TaskID: taskID, TraceID: traceID, Provider: model.Provider,
			ModelID: model.ID, ModelVersion: model.Version, Pricing: model.Pricing,
			Usage: response.TokenUsage, Attempt: attemptNumber, ServiceTier: candidate.ServiceTier,
		})
		if costErr != nil {
			return attempt, nil, false, costErr
		}
	}
	e.appendTrace(ctx, taskID, traceID, candidate, attempt, "")
	return attempt, response, false, nil
}

func classify(err error) (provider.ProviderErrorCode, bool) {
	var providerErr *provider.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Code, providerErr.Retryable || providerErr.Code == provider.ErrProviderUnavailable || providerErr.Code == provider.ErrProviderTimeout || providerErr.Code == provider.ErrRateLimited
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return provider.ErrProviderTimeout, true
	}
	return "", false
}

func (e *Executor) appendTrace(ctx context.Context, taskID, traceID string, candidate domain.RouteCandidate, attempt Attempt, message string) {
	if e.traces == nil {
		return
	}
	ended := attempt.CompletedAt
	status := tracepkg.StatusError
	if attempt.Status == "SUCCEEDED" {
		status = tracepkg.StatusOK
	} else if attempt.Status == "SKIPPED_OPEN_CIRCUIT" {
		status = tracepkg.StatusSkipped
	}
	_ = e.traces.Append(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: traceID, TaskID: taskID,
		SpanID: tracepkg.NewID("span"), Name: "model.generate", Kind: "MODEL_CALL",
		Status: status, Message: message,
		Attributes: map[string]string{
			"model.operation_id": attempt.OperationID, "model.attempt_id": attempt.AttemptID,
			"model.id": candidate.ModelID, "model.version": candidate.ModelVersion,
			"route.class": string(candidate.RouteClass), "service.tier": string(candidate.ServiceTier),
			"inference.effort": string(candidate.InferenceEffort), "attempt.status": attempt.Status,
			"circuit.state": string(attempt.CircuitState),
		},
		StartedAt: attempt.StartedAt, EndedAt: &ended,
	})
}
