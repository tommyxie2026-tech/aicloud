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
	return domain.RouteDecision{
		ID:              fmt.Sprintf("route-%d", now.UnixNano()),
		TaskID:          req.TaskID,
		Selected:        selected,
		Candidates:      candidates,
		Reason:          fmt.Sprintf("selected deployment %s for %s@%s", selected.DeploymentID, selected.ModelID, selected.ModelVersion),
		FallbackChain:   fallback,
		EvidenceVersion: req.EvidenceVersion,
		PolicyVersion:   req.PolicyVersion,
		CreatedAt:       now,
	}, nil
}
