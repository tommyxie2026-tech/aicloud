package cost

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func (l *Ledger) RecordReconciledModelUsage(ctx context.Context, usage ModelUsage, evidence domain.RoutePricingEvidence, policies domain.PricingPolicyRepository) ([]domain.CostEvent, error) {
	if policies == nil {
		return nil, fmt.Errorf("pricing policy repository is required")
	}
	if err := evidence.Validate(); err != nil {
		return nil, err
	}
	policy, err := policies.Get(ctx, evidence.PolicyID, evidence.PolicyVersion)
	if err != nil {
		return nil, fmt.Errorf("load route-time pricing policy: %w", err)
	}
	quoteAt := evidence.Quote.QuotedAt
	quote, err := domain.QuotePricing(policy, domain.PricingUsageEstimate{
		InputTokens:     int64(usage.Usage.InputTokens),
		OutputTokens:    int64(usage.Usage.OutputTokens),
		ContextTokens:   int64(usage.Usage.TotalTokens),
		Region:          policy.Region,
		Batch:           usage.ServiceTier == domain.TierBatch,
		ServiceTier:     usage.ServiceTier,
	}, quoteAt)
	if err != nil {
		return nil, fmt.Errorf("reconcile route-time pricing policy: %w", err)
	}

	now := l.now().UTC()
	events := make([]domain.CostEvent, 0, len(quote.Components))
	for index, component := range quote.Components {
		event := domain.CostEvent{
			ID:           fmt.Sprintf("cost-%d", now.Add(time.Duration(index)*time.Nanosecond).UnixNano()),
			TaskID:       usage.TaskID,
			TraceID:      usage.TraceID,
			Component:    pricingCostComponent(component.Name),
			Provider:     usage.Provider,
			ModelID:      usage.ModelID,
			ModelVersion: usage.ModelVersion,
			DeploymentID: usage.DeploymentID,
			Quantity:     component.Quantity,
			Unit:         component.Unit,
			UnitPrice:    component.UnitPrice * component.Factor,
			Amount:       component.Amount,
			Currency:     quote.Currency,
			Attempt:      usage.Attempt,
			Metadata: map[string]string{
				"deployment_id":         usage.DeploymentID,
				"route_decision_id":     evidence.RouteDecisionID,
				"pricing_policy_id":     evidence.PolicyID,
				"pricing_policy_version": evidence.PolicyVersion,
				"pricing_policy_digest": evidence.PolicyDigest,
				"pricing_component":     component.Name,
			},
			CreatedAt: now.Add(time.Duration(index) * time.Nanosecond),
		}
		events = append(events, event)
	}
	if l.events == nil {
		return events, nil
	}
	for index, event := range events {
		persisted, err := l.events.Append(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("append reconciled cost event: %w", err)
		}
		events[index] = persisted
	}
	return events, nil
}

func pricingCostComponent(name string) domain.CostComponent {
	switch name {
	case "input-cache-hit":
		return domain.CostModelCache
	case "output":
		return domain.CostModelOutput
	case "capacity", "capacity-fixed":
		return domain.CostServiceTier
	default:
		return domain.CostModelInput
	}
}
