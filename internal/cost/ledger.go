package cost

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

type Ledger struct {
	events domain.CostEventRepository
	now    func() time.Time
}

func New(events domain.CostEventRepository) *Ledger {
	return &Ledger{events: events, now: time.Now}
}

type ModelUsage struct {
	TaskID       string
	TraceID      string
	Provider     string
	ModelID      string
	ModelVersion string
	DeploymentID string
	Pricing      domain.PricingProfile
	Usage        provider.TokenUsage
	Attempt      int
	ServiceTier  domain.ServiceTier
}

func (l *Ledger) RecordModelUsage(ctx context.Context, usage ModelUsage) ([]domain.CostEvent, error) {
	if usage.TaskID == "" || usage.TraceID == "" {
		return nil, fmt.Errorf("task ID and trace ID are required")
	}
	currency := usage.Pricing.Currency
	if currency == "" {
		currency = "USD"
	}
	now := l.now().UTC()
	events := []domain.CostEvent{
		newEvent(now, usage, domain.CostModelInput, float64(usage.Usage.InputTokens), "token", usage.Pricing.InputPerMillion/1_000_000, currency),
		newEvent(now.Add(time.Nanosecond), usage, domain.CostModelOutput, float64(usage.Usage.OutputTokens), "token", usage.Pricing.OutputPerMillion/1_000_000, currency),
	}
	if usage.ServiceTier != "" && usage.Pricing.ServiceTierFactor > 1 {
		base := events[0].Amount + events[1].Amount
		premium := base * (usage.Pricing.ServiceTierFactor - 1)
		events = append(events, domain.CostEvent{
			ID:           fmt.Sprintf("cost-%d", now.Add(2*time.Nanosecond).UnixNano()),
			TaskID:       usage.TaskID,
			TraceID:      usage.TraceID,
			Component:    domain.CostServiceTier,
			Provider:     usage.Provider,
			ModelID:      usage.ModelID,
			ModelVersion: usage.ModelVersion,
			DeploymentID: usage.DeploymentID,
			Quantity:     1,
			Unit:         string(usage.ServiceTier),
			UnitPrice:    premium,
			Amount:       premium,
			Currency:     currency,
			Attempt:      usage.Attempt,
			Metadata:     deploymentMetadata(usage.DeploymentID),
			CreatedAt:    now.Add(2 * time.Nanosecond),
		})
	}
	if l.events == nil {
		return events, nil
	}
	for i, event := range events {
		persisted, err := l.events.Append(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("append cost event: %w", err)
		}
		events[i] = persisted
	}
	return events, nil
}

func (l *Ledger) AggregateTask(ctx context.Context, taskID string) (float64, string, error) {
	if l.events == nil {
		return 0, "", fmt.Errorf("cost event repository is required")
	}
	events, err := l.events.ListByTask(ctx, taskID)
	if err != nil {
		return 0, "", err
	}
	var total float64
	currency := ""
	for _, event := range events {
		if currency == "" {
			currency = event.Currency
		}
		if event.Currency != "" && currency != event.Currency {
			return 0, "", fmt.Errorf("mixed currencies require explicit conversion")
		}
		total += event.Amount
	}
	return total, currency, nil
}

func newEvent(now time.Time, usage ModelUsage, component domain.CostComponent, quantity float64, unit string, unitPrice float64, currency string) domain.CostEvent {
	return domain.CostEvent{
		ID:           fmt.Sprintf("cost-%d", now.UnixNano()),
		TaskID:       usage.TaskID,
		TraceID:      usage.TraceID,
		Component:    component,
		Provider:     usage.Provider,
		ModelID:      usage.ModelID,
		ModelVersion: usage.ModelVersion,
		DeploymentID: usage.DeploymentID,
		Quantity:     quantity,
		Unit:         unit,
		UnitPrice:    unitPrice,
		Amount:       quantity * unitPrice,
		Currency:     currency,
		Attempt:      usage.Attempt,
		Metadata:     deploymentMetadata(usage.DeploymentID),
		CreatedAt:    now,
	}
}

func deploymentMetadata(id string) map[string]string {
	if id == "" {
		return nil
	}
	return map[string]string{"deployment_id": id}
}
