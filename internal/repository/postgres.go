package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresRepositories struct {
	DB             *sql.DB
	Models         *PostgresModels
	Tasks          *PostgresTasks
	RouteDecisions *PostgresRouteDecisions
	CostEvents     *PostgresCostEvents
}

func OpenPostgres(ctx context.Context, dsn string) (*PostgresRepositories, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return &PostgresRepositories{
		DB:             db,
		Models:         &PostgresModels{db: db},
		Tasks:          &PostgresTasks{db: db},
		RouteDecisions: &PostgresRouteDecisions{db: db},
		CostEvents:     &PostgresCostEvents{db: db},
	}, nil
}

type PostgresModels struct{ db *sql.DB }

const modelColumns = `id, name, version, provider, endpoint, deployment_mode,
	lifecycle_state, capabilities, pricing, health_status, health_checked_at,
	p95_latency_ms, error_rate, quota_remaining, capacity_available, queue_depth,
	service_tiers, inference_efforts, evaluation_version, license, license_evidence,
	provenance, artifact_digest, approval_status, risk_level, data_residency,
	created_at, updated_at`

func (r *PostgresModels) List(ctx context.Context) ([]domain.Model, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+modelColumns+` FROM models ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Model, 0)
	for rows.Next() {
		model, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate models: %w", err)
	}
	return items, nil
}

func (r *PostgresModels) Get(ctx context.Context, id string) (domain.Model, error) {
	model, err := scanModel(r.db.QueryRowContext(ctx, `SELECT `+modelColumns+` FROM models WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Model{}, ErrNotFound
	}
	return model, err
}

func (r *PostgresModels) Create(ctx context.Context, model domain.Model) (domain.Model, error) {
	capabilities, pricing, tiers, efforts, licenseEvidence, provenance, err := marshalModelJSON(model)
	if err != nil {
		return domain.Model{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO models (
		id, name, version, provider, endpoint, deployment_mode, lifecycle_state,
		capabilities, pricing, health_status, health_checked_at, p95_latency_ms,
		error_rate, quota_remaining, capacity_available, queue_depth, service_tiers,
		inference_efforts, evaluation_version, license, license_evidence, provenance,
		artifact_digest, approval_status, risk_level, data_residency, created_at, updated_at
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28
	)`, model.ID, model.Name, model.Version, model.Provider, model.Endpoint,
		model.DeploymentMode, model.Lifecycle, capabilities, pricing, model.Health,
		model.HealthCheckedAt, model.P95LatencyMS, model.ErrorRate, model.QuotaRemaining,
		model.CapacityAvailable, model.QueueDepth, tiers, efforts, model.EvaluationVersion,
		model.License, licenseEvidence, provenance, model.ArtifactDigest,
		model.ApprovalStatus, model.RiskLevel, model.DataResidency, model.CreatedAt, model.UpdatedAt)
	if err != nil {
		return domain.Model{}, fmt.Errorf("create model: %w", err)
	}
	return model, nil
}

func (r *PostgresModels) Update(ctx context.Context, model domain.Model) (domain.Model, error) {
	capabilities, pricing, tiers, efforts, licenseEvidence, provenance, err := marshalModelJSON(model)
	if err != nil {
		return domain.Model{}, err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE models SET
		name=$2, version=$3, provider=$4, endpoint=$5, deployment_mode=$6,
		lifecycle_state=$7, capabilities=$8, pricing=$9, health_status=$10,
		health_checked_at=$11, p95_latency_ms=$12, error_rate=$13,
		quota_remaining=$14, capacity_available=$15, queue_depth=$16,
		service_tiers=$17, inference_efforts=$18, evaluation_version=$19,
		license=$20, license_evidence=$21, provenance=$22, artifact_digest=$23,
		approval_status=$24, risk_level=$25, data_residency=$26, updated_at=$27
		WHERE id=$1`, model.ID, model.Name, model.Version, model.Provider, model.Endpoint,
		model.DeploymentMode, model.Lifecycle, capabilities, pricing, model.Health,
		model.HealthCheckedAt, model.P95LatencyMS, model.ErrorRate, model.QuotaRemaining,
		model.CapacityAvailable, model.QueueDepth, tiers, efforts, model.EvaluationVersion,
		model.License, licenseEvidence, provenance, model.ArtifactDigest,
		model.ApprovalStatus, model.RiskLevel, model.DataResidency, model.UpdatedAt)
	if err != nil {
		return domain.Model{}, fmt.Errorf("update model: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return domain.Model{}, err
	}
	return model, nil
}

type scanner interface{ Scan(...any) error }

func scanModel(row scanner) (domain.Model, error) {
	var model domain.Model
	var capabilities, pricing, tiers, efforts, licenseEvidence, provenance []byte
	var healthCheckedAt sql.NullTime
	if err := row.Scan(
		&model.ID, &model.Name, &model.Version, &model.Provider, &model.Endpoint,
		&model.DeploymentMode, &model.Lifecycle, &capabilities, &pricing, &model.Health,
		&healthCheckedAt, &model.P95LatencyMS, &model.ErrorRate, &model.QuotaRemaining,
		&model.CapacityAvailable, &model.QueueDepth, &tiers, &efforts,
		&model.EvaluationVersion, &model.License, &licenseEvidence, &provenance,
		&model.ArtifactDigest, &model.ApprovalStatus, &model.RiskLevel,
		&model.DataResidency, &model.CreatedAt, &model.UpdatedAt,
	); err != nil {
		return domain.Model{}, err
	}
	if healthCheckedAt.Valid {
		value := healthCheckedAt.Time
		model.HealthCheckedAt = &value
	}
	if err := unmarshalJSON(capabilities, &model.Capabilities); err != nil {
		return domain.Model{}, fmt.Errorf("decode model capabilities: %w", err)
	}
	if err := unmarshalJSON(pricing, &model.Pricing); err != nil {
		return domain.Model{}, fmt.Errorf("decode model pricing: %w", err)
	}
	if err := unmarshalJSON(tiers, &model.ServiceTiers); err != nil {
		return domain.Model{}, fmt.Errorf("decode model service tiers: %w", err)
	}
	if err := unmarshalJSON(efforts, &model.InferenceEfforts); err != nil {
		return domain.Model{}, fmt.Errorf("decode model inference efforts: %w", err)
	}
	if err := unmarshalJSON(licenseEvidence, &model.LicenseEvidence); err != nil {
		return domain.Model{}, fmt.Errorf("decode model license evidence: %w", err)
	}
	if err := unmarshalJSON(provenance, &model.Provenance); err != nil {
		return domain.Model{}, fmt.Errorf("decode model provenance: %w", err)
	}
	return model, nil
}

func marshalModelJSON(model domain.Model) ([]byte, []byte, []byte, []byte, []byte, []byte, error) {
	values := []any{model.Capabilities, model.Pricing, model.ServiceTiers, model.InferenceEfforts, model.LicenseEvidence, model.Provenance}
	encoded := make([][]byte, len(values))
	for i, value := range values {
		body, err := json.Marshal(value)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("encode model JSON: %w", err)
		}
		encoded[i] = body
	}
	return encoded[0], encoded[1], encoded[2], encoded[3], encoded[4], encoded[5], nil
}

type PostgresTasks struct{ db *sql.DB }

const taskColumns = `id, agent_id, input, status, result, cost, estimated_cost,
	actual_cost, currency, route_decision_id, trace_id, created_at, updated_at`

func (r *PostgresTasks) List(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+taskColumns+` FROM tasks ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return items, nil
}

func (r *PostgresTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	task, err := scanTask(r.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	return task, err
}

func (r *PostgresTasks) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tasks (
		id, agent_id, input, status, result, cost, estimated_cost, actual_cost,
		currency, route_decision_id, trace_id, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, task.ID,
		task.AgentID, task.Input, task.Status, task.Result, task.Cost,
		task.EstimatedCost, task.ActualCost, task.Currency, task.RouteDecisionID,
		task.TraceID, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

func (r *PostgresTasks) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE tasks SET agent_id=$2, input=$3,
		status=$4, result=$5, cost=$6, estimated_cost=$7, actual_cost=$8,
		currency=$9, route_decision_id=$10, trace_id=$11, updated_at=$12 WHERE id=$1`,
		task.ID, task.AgentID, task.Input, task.Status, task.Result, task.Cost,
		task.EstimatedCost, task.ActualCost, task.Currency, task.RouteDecisionID,
		task.TraceID, task.UpdatedAt)
	if err != nil {
		return domain.Task{}, fmt.Errorf("update task: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func scanTask(row scanner) (domain.Task, error) {
	var task domain.Task
	if err := row.Scan(&task.ID, &task.AgentID, &task.Input, &task.Status,
		&task.Result, &task.Cost, &task.EstimatedCost, &task.ActualCost,
		&task.Currency, &task.RouteDecisionID, &task.TraceID, &task.CreatedAt,
		&task.UpdatedAt); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

type PostgresRouteDecisions struct{ db *sql.DB }

func (r *PostgresRouteDecisions) Create(ctx context.Context, decision domain.RouteDecision) (domain.RouteDecision, error) {
	selected, _ := json.Marshal(decision.Selected)
	candidates, _ := json.Marshal(decision.Candidates)
	fallback, _ := json.Marshal(decision.FallbackChain)
	_, err := r.db.ExecContext(ctx, `INSERT INTO route_decisions (
		id, task_id, selected, candidates, reason, fallback_chain,
		evidence_version, policy_version, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, decision.ID, decision.TaskID,
		selected, candidates, decision.Reason, fallback, decision.EvidenceVersion,
		decision.PolicyVersion, decision.CreatedAt)
	if err != nil {
		return domain.RouteDecision{}, fmt.Errorf("create route decision: %w", err)
	}
	return decision, nil
}

func (r *PostgresRouteDecisions) Get(ctx context.Context, id string) (domain.RouteDecision, error) {
	decision, err := scanRouteDecision(r.db.QueryRowContext(ctx, `SELECT id, task_id,
		selected, candidates, reason, fallback_chain, evidence_version,
		policy_version, created_at FROM route_decisions WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RouteDecision{}, ErrNotFound
	}
	return decision, err
}

func (r *PostgresRouteDecisions) ListByTask(ctx context.Context, taskID string) ([]domain.RouteDecision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_id, selected, candidates,
		reason, fallback_chain, evidence_version, policy_version, created_at
		FROM route_decisions WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list route decisions: %w", err)
	}
	defer rows.Close()
	items := make([]domain.RouteDecision, 0)
	for rows.Next() {
		decision, err := scanRouteDecision(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, decision)
	}
	return items, rows.Err()
}

func scanRouteDecision(row scanner) (domain.RouteDecision, error) {
	var decision domain.RouteDecision
	var selected, candidates, fallback []byte
	if err := row.Scan(&decision.ID, &decision.TaskID, &selected, &candidates,
		&decision.Reason, &fallback, &decision.EvidenceVersion,
		&decision.PolicyVersion, &decision.CreatedAt); err != nil {
		return domain.RouteDecision{}, err
	}
	if err := unmarshalJSON(selected, &decision.Selected); err != nil {
		return domain.RouteDecision{}, err
	}
	if err := unmarshalJSON(candidates, &decision.Candidates); err != nil {
		return domain.RouteDecision{}, err
	}
	if err := unmarshalJSON(fallback, &decision.FallbackChain); err != nil {
		return domain.RouteDecision{}, err
	}
	return decision, nil
}

type PostgresCostEvents struct{ db *sql.DB }

func (r *PostgresCostEvents) Append(ctx context.Context, event domain.CostEvent) (domain.CostEvent, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("encode cost metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO cost_events (
		id, task_id, trace_id, component, provider, model_id, model_version,
		quantity, unit, unit_price, amount, currency, attempt, metadata, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		event.ID, event.TaskID, event.TraceID, event.Component, event.Provider,
		event.ModelID, event.ModelVersion, event.Quantity, event.Unit,
		event.UnitPrice, event.Amount, event.Currency, event.Attempt, metadata,
		event.CreatedAt)
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("append cost event: %w", err)
	}
	return event, nil
}

func (r *PostgresCostEvents) ListByTask(ctx context.Context, taskID string) ([]domain.CostEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_id, trace_id, component,
		provider, model_id, model_version, quantity, unit, unit_price, amount,
		currency, attempt, metadata, created_at FROM cost_events
		WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list cost events: %w", err)
	}
	defer rows.Close()
	items := make([]domain.CostEvent, 0)
	for rows.Next() {
		var event domain.CostEvent
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.TaskID, &event.TraceID,
			&event.Component, &event.Provider, &event.ModelID, &event.ModelVersion,
			&event.Quantity, &event.Unit, &event.UnitPrice, &event.Amount,
			&event.Currency, &event.Attempt, &metadata, &event.CreatedAt); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(metadata, &event.Metadata); err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	return items, rows.Err()
}

func requireAffected(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func unmarshalJSON(body []byte, target any) error {
	if len(body) == 0 || string(body) == "null" {
		return nil
	}
	return json.Unmarshal(body, target)
}
