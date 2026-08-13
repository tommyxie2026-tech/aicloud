package router

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestPlanComputesDecisionWithoutPersistence(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	decisions := repository.NewMemoryRouteDecisions()
	r := New(repository.NewMemoryModels(model("efficient-plan", 1, now)), decisions)
	r.now = func() time.Time { return now }

	decision, err := r.Plan(context.Background(), Request{
		TaskID:                "task-plan",
		RouteClass:            domain.RouteEfficient,
		RequiredCapabilities:  []string{"coding"},
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 100,
		Budget:                1,
		RequireFreshSignals:   true,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.ModelID != "efficient-plan" {
		t.Fatalf("selected model=%q", decision.Selected.ModelID)
	}
	if _, err := decisions.Get(context.Background(), decision.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("planned decision must not be persisted, error=%v", err)
	}
}
