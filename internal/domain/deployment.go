package domain

import (
	"context"
	"time"
)

type DeploymentLifecycle string

const (
	DeploymentDiscovered DeploymentLifecycle = "discovered"
	DeploymentReady      DeploymentLifecycle = "ready"
	DeploymentDegraded   DeploymentLifecycle = "degraded"
	DeploymentDraining   DeploymentLifecycle = "draining"
	DeploymentRetired    DeploymentLifecycle = "retired"
	DeploymentBlocked    DeploymentLifecycle = "blocked"
)

type Deployment struct {
	ID                string              `json:"id"`
	ModelID           string              `json:"modelId"`
	ModelVersion      string              `json:"modelVersion"`
	Provider          string              `json:"provider"`
	Endpoint          string              `json:"endpoint,omitempty"`
	Mode              DeploymentMode      `json:"mode"`
	Region            string              `json:"region,omitempty"`
	DataResidency     string              `json:"dataResidency,omitempty"`
	Runtime           string              `json:"runtime,omitempty"`
	Quantization      string              `json:"quantization,omitempty"`
	PricingPolicyRef  string              `json:"pricingPolicyRef,omitempty"`
	Health            HealthStatus        `json:"health"`
	HealthCheckedAt   *time.Time           `json:"healthCheckedAt,omitempty"`
	P95LatencyMS      int64               `json:"p95LatencyMs,omitempty"`
	ErrorRate         float64             `json:"errorRate,omitempty"`
	QuotaRemaining    int64               `json:"quotaRemaining,omitempty"`
	CapacityAvailable int64               `json:"capacityAvailable,omitempty"`
	QueueDepth        int64               `json:"queueDepth,omitempty"`
	ServiceTiers      []ServiceTier       `json:"serviceTiers,omitempty"`
	InferenceEfforts  []InferenceEffort   `json:"inferenceEfforts,omitempty"`
	Lifecycle         DeploymentLifecycle `json:"lifecycle"`
	RoutingEligible   bool                `json:"routingEligible"`
	Owner             string              `json:"owner,omitempty"`
	PolicyRef         string              `json:"policyRef,omitempty"`
	ReplacementIDs    []string            `json:"replacementIds,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
	UpdatedAt         time.Time           `json:"updatedAt"`
}

func (d Deployment) IsRoutingEligible(now time.Time, maxSignalAge time.Duration) bool {
	if !d.RoutingEligible {
		return false
	}
	if d.Lifecycle != DeploymentReady && d.Lifecycle != DeploymentDegraded {
		return false
	}
	if d.Health == HealthUnhealthy {
		return false
	}
	if maxSignalAge > 0 {
		if d.HealthCheckedAt == nil || now.Sub(*d.HealthCheckedAt) > maxSignalAge {
			return false
		}
	}
	return true
}

type DeploymentRepository interface {
	List(context.Context) ([]Deployment, error)
	ListByModel(context.Context, string, string) ([]Deployment, error)
	Get(context.Context, string) (Deployment, error)
	Create(context.Context, Deployment) (Deployment, error)
	Update(context.Context, Deployment) (Deployment, error)
}
