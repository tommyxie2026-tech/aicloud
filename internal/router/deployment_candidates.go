package router

import (
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func DeploymentCandidate(model domain.Model, deployment domain.Deployment, req Request, now time.Time) domain.RouteCandidate {
	candidate := domain.RouteCandidate{
		ModelID:         model.ID,
		ModelVersion:    model.Version,
		DeploymentID:    deployment.ID,
		RouteClass:      req.RouteClass,
		InferenceEffort: req.InferenceEffort,
		ServiceTier:     req.ServiceTier,
	}
	reject := func(reason string) { candidate.RejectionReasons = append(candidate.RejectionReasons, reason) }

	if model.Lifecycle != domain.ModelActive && !(req.AllowDegraded && model.Lifecycle == domain.ModelDegraded) {
		reject("model lifecycle is not routable")
	}
	if model.ApprovalStatus != domain.ApprovalApproved {
		reject("model version is not approved")
	}
	for _, capability := range req.RequiredCapabilities {
		if !containsFold(model.Capabilities, capability) {
			reject("missing capability: " + capability)
		}
	}

	maxAge := time.Duration(0)
	if req.RequireFreshSignals {
		maxAge = req.SignalMaxAge
	}
	if !deployment.IsRoutingEligible(now, maxAge) {
		reject("deployment is not eligible")
	}
	if deployment.Lifecycle == domain.DeploymentDegraded && !req.AllowDegraded {
		reject("degraded deployment is not allowed")
	}
	if deployment.HealthCheckedAt != nil && deployment.CapacityAvailable == 0 {
		reject("deployment has no available capacity")
	}
	if deployment.HealthCheckedAt != nil && deployment.QuotaRemaining == 0 {
		reject("deployment quota is unavailable")
	}
	if req.DataResidency != "" && deployment.DataResidency != "" && !strings.EqualFold(req.DataResidency, deployment.DataResidency) {
		reject("data residency does not match")
	}
	if req.InferenceEffort != "" && len(deployment.InferenceEfforts) > 0 && !containsEffort(deployment.InferenceEfforts, req.InferenceEffort) {
		reject("inference effort is unsupported")
	}
	if req.ServiceTier != "" && len(deployment.ServiceTiers) > 0 && !containsTier(deployment.ServiceTiers, req.ServiceTier) {
		reject("service tier is unsupported")
	}

	candidate.EstimatedCost = estimateCost(model.Pricing, req.EstimatedInputTokens, req.EstimatedOutputTokens)
	if req.Budget > 0 && candidate.EstimatedCost > req.Budget {
		reject("estimated cost exceeds budget")
	}
	candidate.Score = 50 - candidate.EstimatedCost
	if deployment.Health == domain.HealthHealthy {
		candidate.Score += 20
	}
	if deployment.CapacityAvailable > 0 {
		candidate.Score += 10
	}
	if deployment.P95LatencyMS > 0 {
		candidate.Score -= float64(deployment.P95LatencyMS) / 1000
	}
	return candidate
}
