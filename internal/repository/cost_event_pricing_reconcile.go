package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type routePricingInputs struct {
	ServiceTier     domain.ServiceTier     `json:"serviceTier"`
	InferenceEffort domain.InferenceEffort `json:"inferenceEffort"`
	InputTokens     int64                  `json:"estimatedInputTokens"`
	OutputTokens    int64                  `json:"estimatedOutputTokens"`
}

func reconcileDeploymentCostEvent(ctx context.Context, db *sql.DB, event domain.CostEvent) (domain.CostEvent, error) {
	if db == nil || event.DeploymentID == "" || event.TaskID == "" {
		return event, nil
	}
	var routeID string
	var routeCreatedAt sql.NullTime
	err := db.QueryRowContext(ctx, `SELECT t.route_decision_id, rd.created_at FROM tasks t JOIN route_decisions rd ON rd.id=t.route_decision_id WHERE t.id=$1`, event.TaskID).Scan(&routeID, &routeCreatedAt)
	if errors.Is(err, sql.ErrNoRows) || routeID == "" || !routeCreatedAt.Valid {
		return event, nil
	}
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("load route identity for cost reconciliation: %w", err)
	}

	var policyID, policyVersion, policyDigest string
	err = db.QueryRowContext(ctx, `SELECT policy_id, policy_version, policy_digest FROM route_pricing_evidence WHERE route_decision_id=$1 AND deployment_id=$2`, routeID, event.DeploymentID).Scan(&policyID, &policyVersion, &policyDigest)
	var policy domain.PricingPolicy
	pricing := NewPostgresPricingPolicies(db)
	if err == nil {
		policy, err = pricing.Get(ctx, policyID, policyVersion)
	} else if errors.Is(err, sql.ErrNoRows) {
		policy, err = pricing.Resolve(ctx, event.DeploymentID, routeCreatedAt.Time)
		if err == nil {
			policyID, policyVersion, policyDigest = policy.ID, policy.Version, policy.Digest
		}
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		return event, nil
	}
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("resolve route-time policy for cost reconciliation: %w", err)
	}

	var payload []byte
	inputs := routePricingInputs{}
	if err := db.QueryRowContext(ctx, `SELECT payload FROM task_events WHERE task_id=$1 AND event_type='TaskRoutingStarted' AND payload->>'routeDecisionId'=$2 ORDER BY sequence DESC LIMIT 1`, event.TaskID, routeID).Scan(&payload); err == nil {
		_ = json.Unmarshal(payload, &inputs)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return domain.CostEvent{}, fmt.Errorf("load route pricing inputs: %w", err)
	}

	usage := domain.PricingUsageEstimate{
		ContextTokens:   inputs.InputTokens + inputs.OutputTokens,
		Region:          policy.Region,
		Batch:           inputs.ServiceTier == domain.TierBatch,
		ServiceTier:     inputs.ServiceTier,
		InferenceEffort: inputs.InferenceEffort,
	}
	switch event.Component {
	case domain.CostModelInput:
		usage.InputTokens = int64(event.Quantity)
	case domain.CostModelOutput:
		usage.OutputTokens = int64(event.Quantity)
	case domain.CostModelCache:
		usage.InputTokens = int64(event.Quantity)
		usage.CacheHitInputTokens = int64(event.Quantity)
	case domain.CostServiceTier:
		event.UnitPrice = 0
		event.Amount = 0
	}
	if event.Component == domain.CostModelInput || event.Component == domain.CostModelOutput || event.Component == domain.CostModelCache {
		quote, quoteErr := domain.QuotePricing(policy, usage, routeCreatedAt.Time)
		if quoteErr != nil {
			return domain.CostEvent{}, fmt.Errorf("quote route-time policy for cost reconciliation: %w", quoteErr)
		}
		for _, component := range quote.Components {
			if pricingComponentMatchesCostEvent(component.Name, event.Component) {
				event.UnitPrice = component.UnitPrice * component.Factor
				event.Amount = component.Amount
				break
			}
		}
		event.Currency = quote.Currency
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	event.Metadata["deployment_id"] = event.DeploymentID
	event.Metadata["route_decision_id"] = routeID
	event.Metadata["pricing_policy_id"] = policyID
	event.Metadata["pricing_policy_version"] = policyVersion
	event.Metadata["pricing_policy_digest"] = policyDigest
	return event, nil
}

func pricingComponentMatchesCostEvent(name string, component domain.CostComponent) bool {
	switch component {
	case domain.CostModelInput:
		return name == "input-cache-miss"
	case domain.CostModelOutput:
		return name == "output"
	case domain.CostModelCache:
		return name == "input-cache-hit"
	default:
		return false
	}
}
