package router

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func pricingEvidenceRef(ctx context.Context, repo domain.PricingPolicyRepository, deploymentID string, at time.Time) string {
	if repo == nil || deploymentID == "" {
		return ""
	}
	policy, err := repo.Resolve(ctx, deploymentID, at)
	if err != nil {
		return ""
	}
	ref := fmt.Sprintf("%s@%s", policy.ID, policy.Version)
	if policy.Digest != "" {
		ref += "#" + policy.Digest
	}
	return ref
}
