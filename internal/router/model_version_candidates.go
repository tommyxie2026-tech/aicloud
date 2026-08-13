package router

import (
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func ModelVersionCandidate(version domain.ModelVersion, deployment domain.Deployment, req Request, now time.Time) domain.RouteCandidate {
	compat := version.LegacyModel(deployment.Mode)
	compat.Pricing = deployment.Pricing
	return DeploymentCandidate(compat, deployment, req, now)
}
