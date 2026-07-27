package router

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type Request struct {
	TaskID               string
	RouteClass            domain.RouteClass
	RequiredCapabilities []string
	InferenceEffort       domain.InferenceEffort
	ServiceTier           domain.ServiceTier
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	Budget                float64
	Currency              string
	DataResidency         string
	EvidenceVersion       string
	PolicyVersion         string
	AllowDegraded         bool
	RequireFreshSignals   bool
	SignalMaxAge          time.Duration
}

type Router struct {
	models    domain.ModelRepository
	decisions domain.RouteDecisionRepository
	now       func() time.Time
}

func New(models domain.ModelRepository, decisions domain.RouteDecisionRepository) *Router {
	return &Router{models: models, decisions: decisions, now: time.Now}
}

func (r *Router) Decide(ctx context.Context, req Request) (domain.RouteDecision, error) {
	if req.TaskID == "" {
		return domain.RouteDecision{}, fmt.Errorf("task ID is required")
	}
	if req.RouteClass == "" {
		req.RouteClass = domain.RouteEfficient
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.SignalMaxAge <= 0 {
		req.SignalMaxAge = 5 * time.Minute
	}
	now := r.now().UTC()
	if req.RouteClass == domain.RouteDeterministic {
		decision := domain.RouteDecision{
			ID:              fmt.Sprintf("route-%d", now.UnixNano()),
			TaskID:          req.TaskID,
			Selected:        domain.RouteCandidate{RouteClass: domain.RouteDeterministic, Score: 100},
			Reason:          "task classified for deterministic execution without a model call",
			EvidenceVersion: req.EvidenceVersion,
			PolicyVersion:   req.PolicyVersion,
			CreatedAt:       now,
		}
		return r.persist(ctx, decision)
	}

	models, err := r.models.List(ctx)
	if err != nil {
		return domain.RouteDecision{}, fmt.Errorf("list route models: %w", err)
	}
	candidates := make([]domain.RouteCandidate, 0, len(models))
	eligible := make([]domain.RouteCandidate, 0, len(models))
	for _, model := range models {
		candidate := evaluate(model, req, now)
		candidates = append(candidates, candidate)
		if len(candidate.RejectionReasons) == 0 {
			eligible = append(eligible, candidate)
		}
	}
	if len(eligible) == 0 {
		return domain.RouteDecision{}, fmt.Errorf("no policy-compliant model capacity is available")
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Score == eligible[j].Score {
			return eligible[i].EstimatedCost < eligible[j].EstimatedCost
		}
		return eligible[i].Score > eligible[j].Score
	})
	selected := eligible[0]
	fallback := append([]domain.RouteCandidate(nil), eligible[1:]...)
	decision := domain.RouteDecision{
		ID:              fmt.Sprintf("route-%d", now.UnixNano()),
		TaskID:          req.TaskID,
		Selected:        selected,
		Candidates:      candidates,
		Reason:          fmt.Sprintf("selected %s@%s by capability, governance, health, capacity and estimated task cost", selected.ModelID, selected.ModelVersion),
		FallbackChain:   fallback,
		EvidenceVersion: req.EvidenceVersion,
		PolicyVersion:   req.PolicyVersion,
		CreatedAt:       now,
	}
	return r.persist(ctx, decision)
}

func (r *Router) persist(ctx context.Context, decision domain.RouteDecision) (domain.RouteDecision, error) {
	if r.decisions == nil {
		return decision, nil
	}
	return r.decisions.Create(ctx, decision)
}

func evaluate(model domain.Model, req Request, now time.Time) domain.RouteCandidate {
	candidate := domain.RouteCandidate{
		ModelID:         model.ID,
		ModelVersion:    model.Version,
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
	if model.Health == domain.HealthUnhealthy {
		reject("model endpoint is unhealthy")
	}
	if req.RequireFreshSignals && (model.HealthCheckedAt == nil || now.Sub(*model.HealthCheckedAt) > req.SignalMaxAge) {
		reject("runtime health and capacity signals are stale")
	}
	if model.HealthCheckedAt != nil && model.CapacityAvailable <= 0 {
		reject("model has no reported available capacity")
	}
	if model.HealthCheckedAt != nil && model.QuotaRemaining <= 0 {
		reject("provider quota is exhausted")
	}
	if req.DataResidency != "" && model.DataResidency != "" && !strings.EqualFold(req.DataResidency, model.DataResidency) {
		reject("data-residency requirement is not satisfied")
	}
	for _, capability := range req.RequiredCapabilities {
		if !containsFold(model.Capabilities, capability) {
			reject("missing capability: " + capability)
		}
	}
	if req.InferenceEffort != "" && len(model.InferenceEfforts) > 0 && !containsEffort(model.InferenceEfforts, req.InferenceEffort) {
		reject("requested inference effort is unsupported")
	}
	if req.ServiceTier != "" && len(model.ServiceTiers) > 0 && !containsTier(model.ServiceTiers, req.ServiceTier) {
		reject("requested service tier is unsupported")
	}
	candidate.EstimatedCost = estimateCost(model.Pricing, req.EstimatedInputTokens, req.EstimatedOutputTokens)
	if req.Budget > 0 && candidate.EstimatedCost > req.Budget {
		reject("estimated model cost exceeds task budget")
	}
	candidate.Score = score(model, req, candidate.EstimatedCost)
	return candidate
}

func estimateCost(pricing domain.PricingProfile, inputTokens, outputTokens int) float64 {
	cost := float64(inputTokens)/1_000_000*pricing.InputPerMillion + float64(outputTokens)/1_000_000*pricing.OutputPerMillion
	factor := pricing.ServiceTierFactor
	if factor <= 0 {
		factor = 1
	}
	return cost * factor
}

func score(model domain.Model, req Request, estimatedCost float64) float64 {
	score := 50.0
	if model.Health == domain.HealthHealthy {
		score += 20
	} else if model.Health == domain.HealthDegraded {
		score += 5
	}
	if model.CapacityAvailable > 0 {
		score += 10
	}
	if model.P95LatencyMS > 0 {
		score -= float64(model.P95LatencyMS) / 1000
	}
	score -= estimatedCost
	if req.RouteClass == domain.RouteSpecialist && hasSpecialistCapability(model.Capabilities, req.RequiredCapabilities) {
		score += 15
	}
	return score
}

func hasSpecialistCapability(available, required []string) bool {
	for _, capability := range required {
		if containsFold(available, capability) {
			return true
		}
	}
	return false
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func containsEffort(items []domain.InferenceEffort, target domain.InferenceEffort) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsTier(items []domain.ServiceTier, target domain.ServiceTier) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
