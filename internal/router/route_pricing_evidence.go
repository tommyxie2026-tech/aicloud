package router

import (
	"context"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func (r *Router) PricingEvidence(ctx context.Context, decision domain.RouteDecision, req Request) ([]domain.RoutePricingEvidence, error) {
	deployments := r.deploymentRepository()
	pricing := r.pricingPolicyRepository()
	if deployments == nil || pricing == nil {
		return nil, nil
	}
	at := decision.CreatedAt
	if at.IsZero() {
		return nil, fmt.Errorf("route decision creation time is required for pricing replay evidence")
	}
	evidence := make([]domain.RoutePricingEvidence, 0, len(decision.Candidates))
	for _, candidate := range decision.Candidates {
		if candidate.DeploymentID == "" || len(candidate.RejectionReasons) > 0 {
			continue
		}
		deployment, err := deployments.Get(ctx, candidate.DeploymentID)
		if err != nil {
			if candidate.DeploymentID == decision.Selected.DeploymentID {
				return nil, fmt.Errorf("load selected deployment pricing evidence: %w", err)
			}
			continue
		}
		policy, err := pricing.Resolve(ctx, candidate.DeploymentID, at)
		if err != nil {
			if candidate.DeploymentID == decision.Selected.DeploymentID {
				return nil, fmt.Errorf("resolve selected deployment pricing evidence: %w", err)
			}
			continue
		}
		quote, err := domain.QuotePricing(policy, domain.PricingUsageEstimate{
			InputTokens:     int64(req.EstimatedInputTokens),
			OutputTokens:    int64(req.EstimatedOutputTokens),
			ContextTokens:   int64(req.EstimatedInputTokens + req.EstimatedOutputTokens),
			Region:          deployment.Region,
			Batch:           req.ServiceTier == domain.TierBatch,
			ServiceTier:     req.ServiceTier,
			InferenceEffort: req.InferenceEffort,
		}, at)
		if err != nil {
			if candidate.DeploymentID == decision.Selected.DeploymentID {
				return nil, fmt.Errorf("quote selected deployment pricing evidence: %w", err)
			}
			continue
		}
		item := domain.RoutePricingEvidence{
			RouteDecisionID: decision.ID,
			DeploymentID:    candidate.DeploymentID,
			PolicyID:        policy.ID,
			PolicyVersion:   policy.Version,
			PolicyDigest:    policy.Digest,
			Quote:           quote,
			Selected:        candidate.DeploymentID == decision.Selected.DeploymentID,
			CreatedAt:       at,
		}
		if err := item.Validate(); err != nil {
			return nil, err
		}
		evidence = append(evidence, item)
	}
	return evidence, nil
}
