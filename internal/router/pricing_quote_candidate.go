package router

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func quoteCandidate(ctx context.Context, repo domain.PricingPolicyRepository, candidate domain.RouteCandidate, deployment domain.Deployment, req Request, at time.Time) (domain.RouteCandidate, string) {
	if repo == nil || candidate.DeploymentID == "" {
		return candidate, ""
	}
	policy, err := repo.Resolve(ctx, candidate.DeploymentID, at)
	if err != nil {
		candidate.RejectionReasons = append(candidate.RejectionReasons, "effective pricing policy is unavailable")
		return candidate, ""
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
		candidate.RejectionReasons = append(candidate.RejectionReasons, "pricing policy cannot quote requested workload")
		return candidate, ""
	}
	previousEstimate := candidate.EstimatedCost
	candidate.EstimatedCost = quote.Total
	candidate.Score += previousEstimate - quote.Total
	if req.Budget > 0 && quote.Total > req.Budget {
		candidate.RejectionReasons = append(candidate.RejectionReasons, "estimated cost exceeds budget")
	}
	return candidate, fmt.Sprintf("%s@%s", quote.PolicyID, quote.PolicyVersion)
}
