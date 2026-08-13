package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresDeployments struct{ db *sql.DB }

const deploymentColumns = `id, model_version_id, model_id, model_version, provider, endpoint, deployment_mode,
region, data_residency, runtime, quantization, pricing_policy_ref, pricing, health_status,
health_checked_at, p95_latency_ms, error_rate, quota_remaining, capacity_available,
queue_depth, service_tiers, inference_efforts, lifecycle_state, routing_eligible,
owner_name, policy_ref, replacement_ids, created_at, updated_at`

func (r *PostgresDeployments) List(ctx context.Context) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	defer rows.Close()
	return scanDeployments(rows)
}

func (r *PostgresDeployments) ListByModel(ctx context.Context, modelID, version string) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments WHERE model_id=$1 AND model_version=$2 ORDER BY id`, modelID, version)
	if err != nil {
		return nil, fmt.Errorf("list model deployments: %w", err)
	}
	defer rows.Close()
	return scanDeployments(rows)
}

func (r *PostgresDeployments) ListByModelVersion(ctx context.Context, modelVersionID string) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments WHERE model_version_id=$1 ORDER BY id`, modelVersionID)
	if err != nil {
		return nil, fmt.Errorf("list model-version deployments: %w", err)
	}
	defer rows.Close()
	return scanDeployments(rows)
}

func (r *PostgresDeployments) Get(ctx context.Context, id string) (domain.Deployment, error) {
	item, err := scanDeployment(r.db.QueryRowContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Deployment{}, ErrNotFound
	}
	return item, err
}

func (r *PostgresDeployments) Create(ctx context.Context, d domain.Deployment) (domain.Deployment, error) {
	normalizeDeploymentIdentity(&d)
	pricing, _ := json.Marshal(d.Pricing)
	tiers, _ := json.Marshal(d.ServiceTiers)
	efforts, _ := json.Marshal(d.InferenceEfforts)
	replacements, _ := json.Marshal(d.ReplacementIDs)
	_, err := r.db.ExecContext(ctx, `INSERT INTO model_deployments (`+deploymentColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)`,
		d.ID, d.ModelVersionID, d.ModelID, d.ModelVersion, d.Provider, d.Endpoint, d.Mode,
		d.Region, d.DataResidency, d.Runtime, d.Quantization, d.PricingPolicyRef, pricing,
		d.Health, d.HealthCheckedAt, d.P95LatencyMS, d.ErrorRate, d.QuotaRemaining,
		d.CapacityAvailable, d.QueueDepth, tiers, efforts, d.Lifecycle, d.RoutingEligible,
		d.Owner, d.PolicyRef, replacements, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("create deployment: %w", err)
	}
	return d, nil
}

func (r *PostgresDeployments) Update(ctx context.Context, d domain.Deployment) (domain.Deployment, error) {
	normalizeDeploymentIdentity(&d)
	pricing, _ := json.Marshal(d.Pricing)
	tiers, _ := json.Marshal(d.ServiceTiers)
	efforts, _ := json.Marshal(d.InferenceEfforts)
	replacements, _ := json.Marshal(d.ReplacementIDs)
	result, err := r.db.ExecContext(ctx, `UPDATE model_deployments SET model_version_id=$2,model_id=$3,model_version=$4,provider=$5,endpoint=$6,deployment_mode=$7,region=$8,data_residency=$9,runtime=$10,quantization=$11,pricing_policy_ref=$12,pricing=$13,health_status=$14,health_checked_at=$15,p95_latency_ms=$16,error_rate=$17,quota_remaining=$18,capacity_available=$19,queue_depth=$20,service_tiers=$21,inference_efforts=$22,lifecycle_state=$23,routing_eligible=$24,owner_name=$25,policy_ref=$26,replacement_ids=$27,updated_at=$28 WHERE id=$1`,
		d.ID, d.ModelVersionID, d.ModelID, d.ModelVersion, d.Provider, d.Endpoint, d.Mode,
		d.Region, d.DataResidency, d.Runtime, d.Quantization, d.PricingPolicyRef, pricing,
		d.Health, d.HealthCheckedAt, d.P95LatencyMS, d.ErrorRate, d.QuotaRemaining,
		d.CapacityAvailable, d.QueueDepth, tiers, efforts, d.Lifecycle, d.RoutingEligible,
		d.Owner, d.PolicyRef, replacements, d.UpdatedAt)
	if err != nil {
		return domain.Deployment{}, fmt.Errorf("update deployment: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return domain.Deployment{}, err
	}
	return d, nil
}

func scanDeployments(rows *sql.Rows) ([]domain.Deployment, error) {
	items := []domain.Deployment{}
	for rows.Next() {
		item, err := scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDeployment(row scanner) (domain.Deployment, error) {
	var d domain.Deployment
	var pricing, tiers, efforts, replacements []byte
	if err := row.Scan(
		&d.ID, &d.ModelVersionID, &d.ModelID, &d.ModelVersion, &d.Provider, &d.Endpoint,
		&d.Mode, &d.Region, &d.DataResidency, &d.Runtime, &d.Quantization,
		&d.PricingPolicyRef, &pricing, &d.Health, &d.HealthCheckedAt, &d.P95LatencyMS,
		&d.ErrorRate, &d.QuotaRemaining, &d.CapacityAvailable, &d.QueueDepth, &tiers,
		&efforts, &d.Lifecycle, &d.RoutingEligible, &d.Owner, &d.PolicyRef,
		&replacements, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return domain.Deployment{}, err
	}
	if err := json.Unmarshal(pricing, &d.Pricing); err != nil {
		return domain.Deployment{}, err
	}
	if err := json.Unmarshal(tiers, &d.ServiceTiers); err != nil {
		return domain.Deployment{}, err
	}
	if err := json.Unmarshal(efforts, &d.InferenceEfforts); err != nil {
		return domain.Deployment{}, err
	}
	if err := json.Unmarshal(replacements, &d.ReplacementIDs); err != nil {
		return domain.Deployment{}, err
	}
	return d, nil
}
