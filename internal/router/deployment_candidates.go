package router

import (
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func DeploymentCandidate(model domain.Model, deployment domain.Deployment, req Request, now time.Time) domain.RouteCandidate {
	candidate := domain.RouteCandidate{
		ModelID: model.ID,
		ModelVersion: model.Version,
		DeploymentID: deployment.ID,
		RouteClass: req.RouteClass,
		InferenceEffort: req.InferenceEffort,
		ServiceTier: req.ServiceTier,
	}
	maxAge := time.Duration(0)
	if req.RequireFreshSignals {
		maxAge = req.SignalMaxAge
	}
	if !deployment.IsRoutingEligible(now, maxAge) {
		candidate.RejectionReasons = append(candidate.RejectionReasons, "deployment is not eligible")
	}
	return candidate
}
