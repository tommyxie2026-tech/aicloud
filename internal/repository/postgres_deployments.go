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

const deploymentColumns = `id, model_id, model_version, provider, endpoint, deployment_mode,
region, data_residency, runtime, quantization, pricing_policy_ref, health_status,
health_checked_at, p95_latency_ms, error_rate, quota_remaining, capacity_available,
queue_depth, service_tiers, inference_efforts, lifecycle_state, routing_eligible,
owner_name, policy_ref, replacement_ids, created_at, updated_at`

func (r *PostgresDeployments) List(ctx context.Context) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments ORDER BY id`)
	if err != nil { return nil, fmt.Errorf("list deployments: %w", err) }
	defer rows.Close()
	return scanDeployments(rows)
}

func (r *PostgresDeployments) ListByModel(ctx context.Context, modelID, version string) ([]domain.Deployment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments WHERE model_id=$1 AND model_version=$2 ORDER BY id`, modelID, version)
	if err != nil { return nil, fmt.Errorf("list model deployments: %w", err) }
	defer rows.Close()
	return scanDeployments(rows)
}

func (r *PostgresDeployments) Get(ctx context.Context, id string) (domain.Deployment, error) {
	item, err := scanDeployment(r.db.QueryRowContext(ctx, `SELECT `+deploymentColumns+` FROM model_deployments WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) { return domain.Deployment{}, ErrNotFound }
	return item, err
}

func (r *PostgresDeployments) Create(ctx context.Context, d domain.Deployment) (domain.Deployment, error) {
	tiers, _ := json.Marshal(d.ServiceTiers)
	efforts, _ := json.Marshal(d.InferenceEfforts)
	replacements, _ := json.Marshal(d.ReplacementIDs)
	_, err := r.db.ExecContext(ctx, `INSERT INTO model_deployments (`+deploymentColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)`,
		d.ID,d.ModelID,d.ModelVersion,d.Provider,d.Endpoint,d.Mode,d.Region,d.DataResidency,d.Runtime,d.Quantization,d.PricingPolicyRef,d.Health,d.HealthCheckedAt,d.P95LatencyMS,d.ErrorRate,d.QuotaRemaining,d.CapacityAvailable,d.QueueDepth,tiers,efforts,d.Lifecycle,d.RoutingEligible,d.Owner,d.PolicyRef,replacements,d.CreatedAt,d.UpdatedAt)
	if err != nil { return domain.Deployment{}, fmt.Errorf("create deployment: %w", err) }
	return d, nil
}

func (r *PostgresDeployments) Update(ctx context.Context, d domain.Deployment) (domain.Deployment, error) {
	tiers, _ := json.Marshal(d.ServiceTiers)
	efforts, _ := json.Marshal(d.InferenceEfforts)
	replacements, _ := json.Marshal(d.ReplacementIDs)
	result, err := r.db.ExecContext(ctx, `UPDATE model_deployments SET model_id=$2,model_version=$3,provider=$4,endpoint=$5,deployment_mode=$6,region=$7,data_residency=$8,runtime=$9,quantization=$10,pricing_policy_ref=$11,health_status=$12,health_checked_at=$13,p95_latency_ms=$14,error_rate=$15,quota_remaining=$16,capacity_available=$17,queue_depth=$18,service_tiers=$19,inference_efforts=$20,lifecycle_state=$21,routing_eligible=$22,owner_name=$23,policy_ref=$24,replacement_ids=$25,updated_at=$26 WHERE id=$1`,
		d.ID,d.ModelID,d.ModelVersion,d.Provider,d.Endpoint,d.Mode,d.Region,d.DataResidency,d.Runtime,d.Quantization,d.PricingPolicyRef,d.Health,d.HealthCheckedAt,d.P95LatencyMS,d.ErrorRate,d.QuotaRemaining,d.CapacityAvailable,d.QueueDepth,tiers,efforts,d.Lifecycle,d.RoutingEligible,d.Owner,d.PolicyRef,replacements,d.UpdatedAt)
	if err != nil { return domain.Deployment{}, fmt.Errorf("update deployment: %w", err) }
	if err := requireAffected(result); err != nil { return domain.Deployment{}, err }
	return d, nil
}

func scanDeployments(rows *sql.Rows) ([]domain.Deployment, error) {
	items := []domain.Deployment{}
	for rows.Next() {
		item, err := scanDeployment(rows)
		if err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDeployment(row scanner) (domain.Deployment, error) {
	var d domain.Deployment
	var tiers, efforts, replacements []byte
	if err := row.Scan(&d.ID,&d.ModelID,&d.ModelVersion,&d.Provider,&d.Endpoint,&d.Mode,&d.Region,&d.DataResidency,&d.Runtime,&d.Quantization,&d.PricingPolicyRef,&d.Health,&d.HealthCheckedAt,&d.P95LatencyMS,&d.ErrorRate,&d.QuotaRemaining,&d.CapacityAvailable,&d.QueueDepth,&tiers,&efforts,&d.Lifecycle,&d.RoutingEligible,&d.Owner,&d.PolicyRef,&replacements,&d.CreatedAt,&d.UpdatedAt); err != nil { return domain.Deployment{}, err }
	if err := json.Unmarshal(tiers, &d.ServiceTiers); err != nil { return domain.Deployment{}, err }
	if err := json.Unmarshal(efforts, &d.InferenceEfforts); err != nil { return domain.Deployment{}, err }
	if err := json.Unmarshal(replacements, &d.ReplacementIDs); err != nil { return domain.Deployment{}, err }
	return d, nil
}
