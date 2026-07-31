package modelruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/circuitbreaker"
	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func TestExecutorFallsBackOnRetryableProviderFailure(t *testing.T) {
	primary := &fakeProvider{err: provider.NewProviderError(provider.ErrProviderUnavailable, "unavailable", true, nil)}
	secondary := &fakeProvider{response: &provider.ProviderResponse{
		RequestID: "request-1", ProviderName: "secondary", Model: "secondary-model",
		RawText: "ok", TokenUsage: provider.TokenUsage{InputTokens: 100, OutputTokens: 20},
	}}
	providers := NewMemoryProviderRegistry()
	_ = providers.Put(context.Background(), "primary", primary)
	_ = providers.Put(context.Background(), "secondary", secondary)
	models := repository.NewMemoryModels(
		runtimeModel("primary", "primary-provider", 10),
		runtimeModel("secondary", "secondary-provider", 1),
	)
	costEvents := repository.NewMemoryCostEvents()
	traceStore := tracepkg.NewMemoryStore()
	executor := NewExecutor(
		providers,
		models,
		circuitbreaker.New(circuitbreaker.NewMemoryStore(), 1, time.Minute),
		cost.New(costEvents),
		traceStore,
	)
	decision := domain.RouteDecision{
		Selected: domain.RouteCandidate{ModelID: "primary", ModelVersion: "v1", RouteClass: domain.RouteFlagship},
		FallbackChain: []domain.RouteCandidate{{ModelID: "secondary", ModelVersion: "v1", RouteClass: domain.RouteEfficient}},
	}
	result, err := executor.Execute(context.Background(), "task-1", "trace-1", decision, provider.ProviderRequest{RequestID: "request-1", Prompt: "test", OutputSchemaRef: provider.OutputSchemaRef{Name: "result"}})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Fallback || result.Candidate.ModelID != "secondary" || len(result.Attempts) != 2 {
		t.Fatalf("unexpected fallback result: %#v", result)
	}
	if primary.calls != 1 || secondary.calls != 1 {
		t.Fatalf("provider calls primary=%d secondary=%d", primary.calls, secondary.calls)
	}
	events, _ := costEvents.ListByTask(context.Background(), "task-1")
	if len(events) != 2 || events[0].Attempt != 2 || events[1].Attempt != 2 {
		t.Fatalf("successful fallback costs not retained: %#v", events)
	}
	traces, _ := traceStore.ListByTask(context.Background(), "task-1")
	if len(traces) != 2 || traces[0].Status != tracepkg.StatusError || traces[1].Status != tracepkg.StatusOK {
		t.Fatalf("unexpected trace evidence: %#v", traces)
	}
}

func TestExecutorDoesNotFallbackOnNonRetryableFailure(t *testing.T) {
	primaryErr := provider.NewProviderError(provider.ErrSchemaInvalid, "schema invalid", false, errors.New("bad schema"))
	primary := &fakeProvider{err: primaryErr}
	secondary := &fakeProvider{response: &provider.ProviderResponse{RawText: "must not run"}}
	providers := NewMemoryProviderRegistry()
	_ = providers.Put(context.Background(), "primary", primary)
	_ = providers.Put(context.Background(), "secondary", secondary)
	executor := NewExecutor(
		providers,
		repository.NewMemoryModels(runtimeModel("primary", "primary", 1), runtimeModel("secondary", "secondary", 1)),
		circuitbreaker.New(circuitbreaker.NewMemoryStore(), 1, time.Minute),
		nil,
		tracepkg.NewMemoryStore(),
	)
	decision := domain.RouteDecision{
		Selected: domain.RouteCandidate{ModelID: "primary", ModelVersion: "v1"},
		FallbackChain: []domain.RouteCandidate{{ModelID: "secondary", ModelVersion: "v1"}},
	}
	result, err := executor.Execute(context.Background(), "task-2", "trace-2", decision, provider.ProviderRequest{RequestID: "request-2", Prompt: "test", OutputSchemaRef: provider.OutputSchemaRef{Name: "result"}})
	if err == nil {
		t.Fatal("expected non-retryable error")
	}
	if result.Fallback || len(result.Attempts) != 1 || secondary.calls != 0 {
		t.Fatalf("non-retryable failure incorrectly fell back: %#v", result)
	}
}

type fakeProvider struct {
	response *provider.ProviderResponse
	err      error
	calls    int
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Type() provider.ProviderType { return provider.ProviderTypePublic }

func (f *fakeProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{SupportsStructuredOutput: true}
}

func (f *fakeProvider) Generate(context.Context, provider.ProviderRequest) (*provider.ProviderResponse, error) {
	f.calls++
	return f.response, f.err
}

func (f *fakeProvider) Health(context.Context) (*provider.HealthStatus, error) {
	return &provider.HealthStatus{Available: true}, nil
}

func runtimeModel(id, providerName string, price float64) domain.Model {
	return domain.Model{
		ID: id, Version: "v1", Provider: providerName,
		Pricing: domain.PricingProfile{Currency: "USD", InputPerMillion: price, OutputPerMillion: price},
	}
}
