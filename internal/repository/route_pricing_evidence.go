package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type PostgresRoutePricingEvidence struct{ db *sql.DB }

func NewPostgresRoutePricingEvidence(db *sql.DB) *PostgresRoutePricingEvidence {
	if db == nil {
		return nil
	}
	return &PostgresRoutePricingEvidence{db: db}
}

func (r *PostgresRoutePricingEvidence) ListByRoute(ctx context.Context, routeDecisionID string) ([]domain.RoutePricingEvidence, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT route_decision_id, deployment_id, policy_id, policy_version, policy_digest, quote, selected, created_at FROM route_pricing_evidence WHERE route_decision_id=$1 ORDER BY selected DESC, deployment_id`, routeDecisionID)
	if err != nil {
		return nil, fmt.Errorf("list route pricing evidence: %w", err)
	}
	defer rows.Close()
	items := make([]domain.RoutePricingEvidence, 0)
	for rows.Next() {
		var item domain.RoutePricingEvidence
		var quote []byte
		if err := rows.Scan(&item.RouteDecisionID, &item.DeploymentID, &item.PolicyID, &item.PolicyVersion, &item.PolicyDigest, &quote, &item.Selected, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(quote, &item.Quote); err != nil {
			return nil, fmt.Errorf("decode route pricing quote: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertRoutePricingEvidenceTx(ctx context.Context, tx *sql.Tx, items []domain.RoutePricingEvidence) error {
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return err
		}
		quote, err := json.Marshal(item.Quote)
		if err != nil {
			return fmt.Errorf("encode route pricing quote: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO route_pricing_evidence (route_decision_id, deployment_id, policy_id, policy_version, policy_digest, quote, selected, created_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8)`, item.RouteDecisionID, item.DeploymentID, item.PolicyID, item.PolicyVersion, item.PolicyDigest, string(quote), item.Selected, item.CreatedAt); err != nil {
			return fmt.Errorf("persist route pricing evidence: %w", err)
		}
	}
	return nil
}
