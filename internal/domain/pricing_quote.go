package domain

import (
	"fmt"
	"strings"
	"time"
)

type PricingUsageEstimate struct {
	InputTokens         int64           `json:"inputTokens"`
	OutputTokens        int64           `json:"outputTokens"`
	CacheHitInputTokens int64           `json:"cacheHitInputTokens,omitempty"`
	ContextTokens       int64           `json:"contextTokens,omitempty"`
	Region              string          `json:"region,omitempty"`
	Batch               bool            `json:"batch,omitempty"`
	Async               bool            `json:"async,omitempty"`
	ServiceTier         ServiceTier     `json:"serviceTier,omitempty"`
	InferenceEffort     InferenceEffort `json:"inferenceEffort,omitempty"`
	CapacityUnits       float64         `json:"capacityUnits,omitempty"`
	GPUHours            float64         `json:"gpuHours,omitempty"`
	CPUCoreHours        float64         `json:"cpuCoreHours,omitempty"`
	MemoryGBHours       float64         `json:"memoryGbHours,omitempty"`
	StorageGBHours      float64         `json:"storageGbHours,omitempty"`
}

type PricingQuoteComponent struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unitPrice"`
	Factor    float64 `json:"factor"`
	Amount    float64 `json:"amount"`
}

type PricingQuote struct {
	PolicyID      string                  `json:"policyId"`
	PolicyVersion string                  `json:"policyVersion"`
	PolicyDigest  string                  `json:"policyDigest,omitempty"`
	DeploymentID  string                  `json:"deploymentId"`
	Currency      string                  `json:"currency"`
	Components    []PricingQuoteComponent `json:"components"`
	Total         float64                 `json:"total"`
	QuotedAt      time.Time               `json:"quotedAt"`
}

func QuotePricing(policy PricingPolicy, usage PricingUsageEstimate, at time.Time) (PricingQuote, error) {
	if err := policy.Validate(); err != nil {
		return PricingQuote{}, err
	}
	if at.Before(policy.EffectiveFrom) || (policy.EffectiveTo != nil && !at.Before(*policy.EffectiveTo)) {
		return PricingQuote{}, fmt.Errorf("pricing policy %s is not effective at quote time", policy.Ref())
	}
	if policy.Region != "" && usage.Region != "" && !strings.EqualFold(policy.Region, usage.Region) {
		return PricingQuote{}, fmt.Errorf("pricing policy region %s does not match %s", policy.Region, usage.Region)
	}
	if usage.CacheHitInputTokens < 0 || usage.CacheHitInputTokens > usage.InputTokens {
		return PricingQuote{}, fmt.Errorf("cached input tokens must be within input token count")
	}

	inputPrice := policy.InputPerMillion
	outputPrice := policy.OutputPerMillion
	cacheHitPrice := policy.CacheHitPerMillion
	cacheMissPrice := policy.CacheMissPerMillion
	if cacheMissPrice == 0 {
		cacheMissPrice = inputPrice
	}
	for _, band := range policy.ContextBands {
		if usage.ContextTokens < band.MinTokens || (band.MaxTokens > 0 && usage.ContextTokens > band.MaxTokens) {
			continue
		}
		if band.InputPerMillion != nil {
			inputPrice = *band.InputPerMillion
		}
		if band.OutputPerMillion != nil {
			outputPrice = *band.OutputPerMillion
		}
		if band.CacheHitPerMillion != nil {
			cacheHitPrice = *band.CacheHitPerMillion
		}
		if band.CacheMissPerMillion != nil {
			cacheMissPrice = *band.CacheMissPerMillion
		} else if band.InputPerMillion != nil {
			cacheMissPrice = inputPrice
		}
		break
	}

	factor := 1.0
	if usage.Batch && policy.BatchFactor > 0 {
		factor *= policy.BatchFactor
	}
	if usage.Async && policy.AsyncFactor > 0 {
		factor *= policy.AsyncFactor
	}
	if value := policy.ServiceTierFactors[usage.ServiceTier]; value > 0 {
		factor *= value
	}
	if value := policy.InferenceEffortFactors[usage.InferenceEffort]; value > 0 {
		factor *= value
	}

	cacheMissTokens := usage.InputTokens - usage.CacheHitInputTokens
	components := make([]PricingQuoteComponent, 0, 8)
	add := func(name string, quantity float64, unit string, unitPrice, componentFactor float64) {
		if quantity == 0 && unitPrice == 0 {
			return
		}
		components = append(components, PricingQuoteComponent{
			Name: name, Quantity: quantity, Unit: unit, UnitPrice: unitPrice,
			Factor: componentFactor, Amount: quantity * unitPrice * componentFactor,
		})
	}
	add("input-cache-miss", float64(cacheMissTokens)/1_000_000, "million-token", cacheMissPrice, factor)
	add("input-cache-hit", float64(usage.CacheHitInputTokens)/1_000_000, "million-token", cacheHitPrice, factor)
	add("output", float64(usage.OutputTokens)/1_000_000, "million-token", outputPrice, factor)
	add("capacity", usage.CapacityUnits, policy.Capacity.Unit, policy.Capacity.UnitPrice, 1)
	if policy.Capacity.FixedPrice != 0 {
		add("capacity-fixed", 1, "allocation", policy.Capacity.FixedPrice, 1)
	}
	add("self-hosted-gpu", usage.GPUHours, "gpu-hour", policy.SelfHosted.GPUHourPrice, 1)
	add("self-hosted-cpu", usage.CPUCoreHours, "cpu-core-hour", policy.SelfHosted.CPUCoreHourPrice, 1)
	add("self-hosted-memory", usage.MemoryGBHours, "memory-gb-hour", policy.SelfHosted.MemoryGBHourPrice, 1)
	add("self-hosted-storage", usage.StorageGBHours, "storage-gb-hour", policy.SelfHosted.StorageGBHourPrice, 1)

	quote := PricingQuote{
		PolicyID: policy.ID, PolicyVersion: policy.Version, PolicyDigest: policy.Digest,
		DeploymentID: policy.DeploymentID, Currency: policy.Currency,
		Components: components, QuotedAt: at,
	}
	for _, component := range components {
		quote.Total += component.Amount
	}
	return quote, nil
}
