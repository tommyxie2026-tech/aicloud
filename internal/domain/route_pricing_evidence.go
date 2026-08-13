package domain

import (
	"context"
	"fmt"
	"time"
)

type RoutePricingEvidence struct {
	RouteDecisionID string       `json:"routeDecisionId"`
	DeploymentID    string       `json:"deploymentId"`
	PolicyID        string       `json:"policyId"`
	PolicyVersion   string       `json:"policyVersion"`
	PolicyDigest    string       `json:"policyDigest,omitempty"`
	Quote           PricingQuote `json:"quote"`
	Selected        bool         `json:"selected"`
	CreatedAt       time.Time    `json:"createdAt"`
}

func (e RoutePricingEvidence) Validate() error {
	if e.RouteDecisionID == "" || e.DeploymentID == "" {
		return fmt.Errorf("route decision ID and deployment ID are required")
	}
	if e.PolicyID == "" || e.PolicyVersion == "" {
		return fmt.Errorf("pricing policy ID and version are required")
	}
	if e.Quote.DeploymentID != e.DeploymentID || e.Quote.PolicyID != e.PolicyID || e.Quote.PolicyVersion != e.PolicyVersion {
		return fmt.Errorf("pricing quote identity must match route pricing evidence")
	}
	if e.Quote.QuotedAt.IsZero() || e.CreatedAt.IsZero() {
		return fmt.Errorf("pricing quote and evidence timestamps are required")
	}
	return nil
}

type RoutePricingEvidenceRepository interface {
	ListByRoute(context.Context, string) ([]RoutePricingEvidence, error)
}
