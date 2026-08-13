package domain

import (
	"context"
	"fmt"
	"time"
)

type DeploymentLifecycleEvent struct {
	ID              string              `json:"id"`
	DeploymentID    string              `json:"deploymentId"`
	From            DeploymentLifecycle `json:"from"`
	To              DeploymentLifecycle `json:"to"`
	AnnouncedAt     *time.Time          `json:"announcedAt,omitempty"`
	EffectiveAt     time.Time           `json:"effectiveAt"`
	EvidenceRef     string              `json:"evidenceRef,omitempty"`
	ReplacementIDs  []string            `json:"replacementIds,omitempty"`
	QuotaRemaining  *int64              `json:"quotaRemaining,omitempty"`
	RateLimitRef    string              `json:"rateLimitRef,omitempty"`
	RoutingEligible bool                `json:"routingEligible"`
	MigrationState  string              `json:"migrationState,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
}

func (e DeploymentLifecycleEvent) Validate() error {
	if e.ID == "" || e.DeploymentID == "" {
		return fmt.Errorf("deployment lifecycle event ID and deployment ID are required")
	}
	if e.EffectiveAt.IsZero() {
		return fmt.Errorf("deployment lifecycle effective time is required")
	}
	if e.AnnouncedAt != nil && e.AnnouncedAt.After(e.EffectiveAt) {
		return fmt.Errorf("deployment lifecycle announcement cannot follow effective time")
	}
	return ValidateDeploymentTransition(e.From, e.To)
}

type DeploymentLifecycleEventRepository interface {
	Append(context.Context, DeploymentLifecycleEvent) (DeploymentLifecycleEvent, error)
	ListByDeployment(context.Context, string) ([]DeploymentLifecycleEvent, error)
}
