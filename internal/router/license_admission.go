package router

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	licensepolicy "github.com/tommyxie2026-tech/aicloud/internal/policy"
)

func evaluateRouteLicense(ctx context.Context, repo domain.LicenseEvidenceVersionRepository, model domain.Model, deployment domain.Deployment, at time.Time) (string, []string) {
	if repo == nil {
		return "", []string{"versioned license evidence repository is unavailable"}
	}
	evidence, err := repo.Resolve(ctx, model.ID, at)
	if err != nil {
		return "", []string{"approved versioned license evidence is unavailable"}
	}
	use := licensepolicy.LicenseUseContext{
		Commercial:    true,
		HostedService: deployment.Mode == domain.DeploymentPublicAPI || deployment.Mode == domain.DeploymentEnterpriseAPI || deployment.Mode == domain.DeploymentPrivateEndpoint,
		Geography:     deployment.Region,
		At:            at,
	}
	decision := licensepolicy.EvaluateLicense(evidence, use)
	if !decision.Allowed {
		return evidence.Ref(), append([]string{fmt.Sprintf("license evidence %s rejected candidate", evidence.Ref())}, decision.Reasons...)
	}
	return evidence.Ref(), nil
}

func combineEvidenceVersion(existing, licenseRef string) string {
	if licenseRef == "" {
		return existing
	}
	if existing == "" {
		return "license:" + licenseRef
	}
	return existing + ";license:" + licenseRef
}
