package domain

import (
	"context"
	"time"
)

type DeploymentLifecycleEvent struct {
	ID              string              `json:"id"`
	DeploymentID    string              `json:"deploymentId"`
	From            DeploymentLifecycle `json:"from"`
	To              DeploymentLifecycle `json:"to"`
	AnnouncedAt     *time.Time           `json:"announcedAt,omitempty"`
	EffectiveAt     time.Time            `json:"effectiveAt"`
	EvidenceRef     string               `json:"evidenceRef,omitempty"`
	ReplacementIDs  []string             `json:"replacementIds,omitempty"`
	QuotaRemaining  *int64               `json:"quotaRemaining,omitempty"`
	RateLimitRef    string               `json:"rateLimitRef,omitempty"`
	RoutingEligible bool                 `json:"routingEligible"`
	MigrationState  string               `json:"migrationState,omitempty"`
	CreatedAt       time.Time            `json:"createdAt"`
}

type DeploymentLifecycleEventRepository interface {
	Append(context.Context, DeploymentLifecycleEvent) (DeploymentLifecycleEvent, error)
	ListByDeployment(context.Context, string) ([]DeploymentLifecycleEvent, error)
}
