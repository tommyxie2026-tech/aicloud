package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresDeploymentCostEvents struct{ db *sql.DB }

func (r *PostgresRepositories) DeploymentCostEvents() *PostgresDeploymentCostEvents {
	if r == nil || r.DB == nil {
		return nil
	}
	return &PostgresDeploymentCostEvents{db: r.DB}
}

func (r *PostgresDeploymentCostEvents) Append(ctx context.Context, event domain.CostEvent) (domain.CostEvent, error) {
	if r == nil || r.db == nil {
		return domain.CostEvent{}, fmt.Errorf("database is required")
	}
	reconciled, err := reconcileDeploymentCostEvent(ctx, r.db, event)
	if err != nil {
		return domain.CostEvent{}, err
	}
	event = reconciled
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("encode cost metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO cost_events (
		id, task_id, trace_id, component, provider, model_id, model_version,
		deployment_id, quantity, unit, unit_price, amount, currency, attempt,
		metadata, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		event.ID, event.TaskID, event.TraceID, event.Component, event.Provider,
		event.ModelID, event.ModelVersion, event.DeploymentID, event.Quantity,
		event.Unit, event.UnitPrice, event.Amount, event.Currency, event.Attempt,
		metadata, event.CreatedAt)
	if err != nil {
		return domain.CostEvent{}, fmt.Errorf("append deployment cost event: %w", err)
	}
	return event, nil
}

func (r *PostgresDeploymentCostEvents) ListByTask(ctx context.Context, taskID string) ([]domain.CostEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_id, trace_id, component,
		provider, model_id, model_version, deployment_id, quantity, unit,
		unit_price, amount, currency, attempt, metadata, created_at
		FROM cost_events WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list deployment cost events: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CostEvent, 0)
	for rows.Next() {
		var event domain.CostEvent
		var metadata []byte
		if err := rows.Scan(
			&event.ID, &event.TaskID, &event.TraceID, &event.Component,
			&event.Provider, &event.ModelID, &event.ModelVersion, &event.DeploymentID,
			&event.Quantity, &event.Unit, &event.UnitPrice, &event.Amount,
			&event.Currency, &event.Attempt, &metadata, &event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(metadata, &event.Metadata); err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
