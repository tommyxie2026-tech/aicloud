package domain

import (
	"context"
	"fmt"
	"time"
)

type PricingContextBand struct {
	MinTokens            int64    `json:"minTokens"`
	MaxTokens            int64    `json:"maxTokens,omitempty"`
	InputPerMillion      *float64 `json:"inputPerMillion,omitempty"`
	OutputPerMillion     *float64 `json:"outputPerMillion,omitempty"`
	CacheHitPerMillion   *float64 `json:"cacheHitPerMillion,omitempty"`
	CacheMissPerMillion  *float64 `json:"cacheMissPerMillion,omitempty"`
}

type CapacityPricing struct {
	Mode       string  `json:"mode,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	UnitPrice  float64 `json:"unitPrice,omitempty"`
	FixedPrice float64 `json:"fixedPrice,omitempty"`
}

type SelfHostedAllocation struct {
	GPUHourPrice       float64 `json:"gpuHourPrice,omitempty"`
	CPUCoreHourPrice   float64 `json:"cpuCoreHourPrice,omitempty"`
	MemoryGBHourPrice  float64 `json:"memoryGbHourPrice,omitempty"`
	StorageGBHourPrice float64 `json:"storageGbHourPrice,omitempty"`
}

type PricingPolicy struct {
	ID                     string                      `json:"id"`
	Version                string                      `json:"version"`
	DeploymentID           string                      `json:"deploymentId"`
	Currency               string                      `json:"currency"`
	Region                 string                      `json:"region,omitempty"`
	InputPerMillion        float64                     `json:"inputPerMillion"`
	OutputPerMillion       float64                     `json:"outputPerMillion"`
	CacheHitPerMillion     float64                     `json:"cacheHitPerMillion,omitempty"`
	CacheMissPerMillion    float64                     `json:"cacheMissPerMillion,omitempty"`
	ContextBands           []PricingContextBand        `json:"contextBands,omitempty"`
	BatchFactor            float64                     `json:"batchFactor,omitempty"`
	AsyncFactor            float64                     `json:"asyncFactor,omitempty"`
	ServiceTierFactors     map[ServiceTier]float64     `json:"serviceTierFactors,omitempty"`
	InferenceEffortFactors map[InferenceEffort]float64 `json:"inferenceEffortFactors,omitempty"`
	Capacity               CapacityPricing             `json:"capacity,omitempty"`
	SelfHosted             SelfHostedAllocation        `json:"selfHosted,omitempty"`
	EffectiveFrom          time.Time                   `json:"effectiveFrom"`
	EffectiveTo            *time.Time                  `json:"effectiveTo,omitempty"`
	EvidenceRef            string                      `json:"evidenceRef,omitempty"`
	Digest                 string                      `json:"digest,omitempty"`
	CreatedAt              time.Time                   `json:"createdAt"`
}

func (p PricingPolicy) Ref() string {
	return p.ID + "@" + p.Version
}

func (p PricingPolicy) Validate() error {
	if p.ID == "" || p.Version == "" || p.DeploymentID == "" {
		return fmt.Errorf("pricing policy ID, version and deployment ID are required")
	}
	if p.Currency == "" || p.EffectiveFrom.IsZero() {
		return fmt.Errorf("pricing policy currency and effective time are required")
	}
	if p.EffectiveTo != nil && !p.EffectiveTo.After(p.EffectiveFrom) {
		return fmt.Errorf("pricing policy effective end must follow effective start")
	}
	return nil
}

type PricingPolicyRepository interface {
	Create(context.Context, PricingPolicy) (PricingPolicy, error)
	Get(context.Context, string, string) (PricingPolicy, error)
	ListByDeployment(context.Context, string) ([]PricingPolicy, error)
	Resolve(context.Context, string, time.Time) (PricingPolicy, error)
}
