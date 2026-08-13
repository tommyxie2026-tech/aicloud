package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresPricingPolicies struct{ db *sql.DB }

func NewPostgresPricingPolicies(db *sql.DB) *PostgresPricingPolicies {
	if db == nil {
		return nil
	}
	return &PostgresPricingPolicies{db: db}
}

const pricingPolicyColumns = `id, version, deployment_id, currency, region,
	input_per_million, output_per_million, cache_hit_per_million, cache_miss_per_million,
	context_bands, batch_factor, async_factor, service_tier_factors,
	inference_effort_factors, capacity_pricing, self_hosted_allocation,
	effective_from, effective_to, evidence_ref, digest, created_at`

func (r *PostgresPricingPolicies) Create(ctx context.Context, item domain.PricingPolicy) (domain.PricingPolicy, error) {
	if err := item.Validate(); err != nil {
		return domain.PricingPolicy{}, err
	}
	bands, tiers, efforts, capacity, selfHosted, err := marshalPricingPolicyJSON(item)
	if err != nil {
		return domain.PricingPolicy{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO pricing_policies (`+pricingPolicyColumns+`) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		item.ID, item.Version, item.DeploymentID, item.Currency, item.Region,
		item.InputPerMillion, item.OutputPerMillion, item.CacheHitPerMillion,
		item.CacheMissPerMillion, bands, item.BatchFactor, item.AsyncFactor, tiers,
		efforts, capacity, selfHosted, item.EffectiveFrom, item.EffectiveTo,
		item.EvidenceRef, item.Digest, item.CreatedAt)
	if err != nil {
		return domain.PricingPolicy{}, fmt.Errorf("create pricing policy: %w", err)
	}
	return item, nil
}

func (r *PostgresPricingPolicies) Get(ctx context.Context, id, version string) (domain.PricingPolicy, error) {
	item, err := scanPricingPolicy(r.db.QueryRowContext(ctx, `SELECT `+pricingPolicyColumns+` FROM pricing_policies WHERE id=$1 AND version=$2`, id, version))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PricingPolicy{}, ErrNotFound
	}
	return item, err
}

func (r *PostgresPricingPolicies) ListByDeployment(ctx context.Context, deploymentID string) ([]domain.PricingPolicy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+pricingPolicyColumns+` FROM pricing_policies WHERE deployment_id=$1 ORDER BY effective_from DESC, created_at DESC, version DESC`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list pricing policies: %w", err)
	}
	defer rows.Close()
	items := make([]domain.PricingPolicy, 0)
	for rows.Next() {
		item, err := scanPricingPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresPricingPolicies) Resolve(ctx context.Context, deploymentID string, at time.Time) (domain.PricingPolicy, error) {
	item, err := scanPricingPolicy(r.db.QueryRowContext(ctx, `SELECT `+pricingPolicyColumns+` FROM pricing_policies
		WHERE deployment_id=$1 AND effective_from <= $2 AND (effective_to IS NULL OR effective_to > $2)
		ORDER BY effective_from DESC, created_at DESC, version DESC LIMIT 1`, deploymentID, at))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PricingPolicy{}, ErrNotFound
	}
	return item, err
}

func marshalPricingPolicyJSON(item domain.PricingPolicy) ([]byte, []byte, []byte, []byte, []byte, error) {
	values := []any{item.ContextBands, item.ServiceTierFactors, item.InferenceEffortFactors, item.Capacity, item.SelfHosted}
	encoded := make([][]byte, len(values))
	for i, value := range values {
		body, err := json.Marshal(value)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("encode pricing policy JSON: %w", err)
		}
		encoded[i] = body
	}
	return encoded[0], encoded[1], encoded[2], encoded[3], encoded[4], nil
}

func scanPricingPolicy(row scanner) (domain.PricingPolicy, error) {
	var item domain.PricingPolicy
	var bands, tiers, efforts, capacity, selfHosted []byte
	if err := row.Scan(
		&item.ID, &item.Version, &item.DeploymentID, &item.Currency, &item.Region,
		&item.InputPerMillion, &item.OutputPerMillion, &item.CacheHitPerMillion,
		&item.CacheMissPerMillion, &bands, &item.BatchFactor, &item.AsyncFactor,
		&tiers, &efforts, &capacity, &selfHosted, &item.EffectiveFrom,
		&item.EffectiveTo, &item.EvidenceRef, &item.Digest, &item.CreatedAt,
	); err != nil {
		return domain.PricingPolicy{}, err
	}
	for _, target := range []struct {
		body []byte
		into any
	}{
		{bands, &item.ContextBands},
		{tiers, &item.ServiceTierFactors},
		{efforts, &item.InferenceEffortFactors},
		{capacity, &item.Capacity},
		{selfHosted, &item.SelfHosted},
	} {
		if err := unmarshalJSON(target.body, target.into); err != nil {
			return domain.PricingPolicy{}, err
		}
	}
	return item, nil
}
