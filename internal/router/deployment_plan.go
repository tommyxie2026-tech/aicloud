package router

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func (r *Router) PlanWithDeployments(ctx context.Context, req Request, deployments domain.DeploymentRepository) (domain.RouteDecision, error) {
	if deployments == nil || req.RouteClass == domain.RouteDeterministic {
		planner := *r
		planner.decisions = nil
		return planner.Decide(ctx, req)
	}
	if req.TaskID == "" {
		return domain.RouteDecision{}, fmt.Errorf("task ID is required")
	}
	if req.RouteClass == "" {
		req.RouteClass = domain.RouteEfficient
	}
	if req.SignalMaxAge <= 0 {
		req.SignalMaxAge = 5 * time.Minute
	}
	now := r.now().UTC()
	models, err := r.models.List(ctx)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	pricingPolicies := r.pricingPolicyRepository()
	licenseEvidence := r.licenseEvidenceRepository()
	licenseRefs := make(map[string]string)
	candidates := make([]domain.RouteCandidate, 0)
	for _, model := range models {
		items, err := deployments.ListByModelVersion(ctx, model.ID)
		if err != nil {
			return domain.RouteDecision{}, err
		}
		if len(items) == 0 {
			items, err = deployments.ListByModel(ctx, model.ID, model.Version)
			if err != nil {
				return domain.RouteDecision{}, err
			}
		}
		for _, item := range items {
			candidate := DeploymentCandidate(model, item, req, now)
			if pricingPolicies != nil {
				candidate, _ = quoteCandidate(ctx, pricingPolicies, candidate, item, req, now)
			}
			licenseRef, licenseReasons := evaluateRouteLicense(ctx, licenseEvidence, model, item, now)
			if licenseRef != "" {
				licenseRefs[item.ID] = licenseRef
			}
			candidate.RejectionReasons = append(candidate.RejectionReasons, licenseReasons...)
			if r.admission != nil {
				admissionModel := model
				admissionModel.DeploymentMode = item.Mode
				decision, err := r.admission.Check(ctx, admissionModel)
				if err != nil {
					return domain.RouteDecision{}, err
				}
				if !decision.Allowed {
					candidate.RejectionReasons = append(candidate.RejectionReasons, decision.Reasons...)
				}
			}
			candidates = append(candidates, candidate)
		}
	}
	selected, fallback, err := SelectDeploymentCandidate(candidates)
	if err != nil {
		return domain.RouteDecision{}, err
	}
	selectedLicenseRef := licenseRefs[selected.DeploymentID]
	reason := fmt.Sprintf("selected deployment %s for %s@%s", selected.DeploymentID, selected.ModelID, selected.ModelVersion)
	if pricingRef := pricingEvidenceRef(ctx, pricingPolicies, selected.DeploymentID, now); pricingRef != "" {
		reason = fmt.Sprintf("%s using pricing policy %s with estimated task cost %.6f", reason, pricingRef, selected.EstimatedCost)
	}
	if selectedLicenseRef != "" {
		reason = fmt.Sprintf("%s using license evidence %s", reason, selectedLicenseRef)
	}
	return domain.RouteDecision{
		ID:              fmt.Sprintf("route-%d", now.UnixNano()),
		TaskID:          req.TaskID,
		Selected:        selected,
		Candidates:      candidates,
		Reason:          reason,
		FallbackChain:   fallback,
		EvidenceVersion: combineEvidenceVersion(req.EvidenceVersion, selectedLicenseRef),
		PolicyVersion:   req.PolicyVersion,
		CreatedAt:       now,
	}, nil
}
