package cost

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func TestLedgerRecordsAndAggregatesModelCost(t *testing.T) {
	events := repository.NewMemoryCostEvents()
	ledger := New(events)
	ledger.now = func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) }

	recorded, err := ledger.RecordModelUsage(context.Background(), ModelUsage{
		TaskID:       "task-1",
		TraceID:      "trace-1",
		Provider:     "provider-1",
		ModelID:      "model-1",
		ModelVersion: "v1",
		DeploymentID: "deployment-1",
		Pricing: domain.PricingProfile{
			Currency:          "USD",
			InputPerMillion:   2,
			OutputPerMillion:  10,
			ServiceTierFactor: 1.5,
		},
		Usage:       provider.TokenUsage{InputTokens: 500_000, OutputTokens: 100_000},
		Attempt:     1,
		ServiceTier: domain.TierPriority,
	})
	if err != nil {
		t.Fatalf("RecordModelUsage returned error: %v", err)
	}
	if len(recorded) != 3 {
		t.Fatalf("event count = %d", len(recorded))
	}
	for _, event := range recorded {
		if event.DeploymentID != "deployment-1" || event.Metadata["deployment_id"] != "deployment-1" {
			t.Fatalf("deployment identity not retained: %#v", event)
		}
	}
	total, currency, err := ledger.AggregateTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("AggregateTask returned error: %v", err)
	}
	if currency != "USD" {
		t.Fatalf("currency = %s", currency)
	}
	if math.Abs(total-3) > 0.000001 {
		t.Fatalf("total = %f", total)
	}
}

func TestLedgerKeepsRetryAttemptVisible(t *testing.T) {
	events := repository.NewMemoryCostEvents()
	ledger := New(events)
	_, err := ledger.RecordModelUsage(context.Background(), ModelUsage{
		TaskID:       "task-retry",
		TraceID:      "trace-retry",
		Provider:     "provider-1",
		ModelID:      "model-1",
		ModelVersion: "v1",
		DeploymentID: "deployment-retry",
		Pricing:      domain.PricingProfile{Currency: "USD", InputPerMillion: 1, OutputPerMillion: 1},
		Usage:        provider.TokenUsage{InputTokens: 100, OutputTokens: 10},
		Attempt:      2,
	})
	if err != nil {
		t.Fatalf("RecordModelUsage returned error: %v", err)
	}
	stored, err := events.ListByTask(context.Background(), "task-retry")
	if err != nil {
		t.Fatalf("ListByTask returned error: %v", err)
	}
	if len(stored) != 2 || stored[0].Attempt != 2 || stored[1].Attempt != 2 {
		t.Fatalf("retry attempt not retained: %#v", stored)
	}
	if stored[0].DeploymentID != "deployment-retry" || stored[1].DeploymentID != "deployment-retry" {
		t.Fatalf("deployment identity not retained across retry: %#v", stored)
	}
}
