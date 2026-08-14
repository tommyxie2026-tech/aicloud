package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type MemoryDeploymentLifecycleEvents struct {
	mu     sync.RWMutex
	events []domain.DeploymentLifecycleEvent
}

func NewMemoryDeploymentLifecycleEvents() *MemoryDeploymentLifecycleEvents {
	return &MemoryDeploymentLifecycleEvents{}
}

func (r *MemoryDeploymentLifecycleEvents) Append(_ context.Context, event domain.DeploymentLifecycleEvent) (domain.DeploymentLifecycleEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return event, nil
}

func (r *MemoryDeploymentLifecycleEvents) ListByDeployment(_ context.Context, deploymentID string) ([]domain.DeploymentLifecycleEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.DeploymentLifecycleEvent, 0)
	for _, event := range r.events {
		if event.DeploymentID == deploymentID {
			items = append(items, event)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].EffectiveAt.Before(items[j].EffectiveAt) })
	return items, nil
}

type PostgresDeploymentLifecycleEvents struct{ db *sql.DB }

func NewPostgresDeploymentLifecycleEvents(db *sql.DB) *PostgresDeploymentLifecycleEvents {
	if db == nil {
		return nil
	}
	return &PostgresDeploymentLifecycleEvents{db: db}
}

func (r *PostgresDeploymentLifecycleEvents) Append(ctx context.Context, event domain.DeploymentLifecycleEvent) (domain.DeploymentLifecycleEvent, error) {
	replacements, err := json.Marshal(event.ReplacementIDs)
	if err != nil {
		return domain.DeploymentLifecycleEvent{}, fmt.Errorf("encode deployment replacements: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO deployment_lifecycle_events (
		id, deployment_id, from_state, to_state, announced_at, effective_at,
		evidence_ref, replacement_ids, quota_remaining, rate_limit_ref,
		routing_eligible, migration_state, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		event.ID, event.DeploymentID, event.From, event.To, event.AnnouncedAt,
		event.EffectiveAt, event.EvidenceRef, replacements, event.QuotaRemaining,
		event.RateLimitRef, event.RoutingEligible, event.MigrationState, event.CreatedAt)
	if err != nil {
		return domain.DeploymentLifecycleEvent{}, fmt.Errorf("append deployment lifecycle event: %w", err)
	}
	return event, nil
}

func (r *PostgresDeploymentLifecycleEvents) ListByDeployment(ctx context.Context, deploymentID string) ([]domain.DeploymentLifecycleEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, deployment_id, from_state, to_state,
		announced_at, effective_at, evidence_ref, replacement_ids, quota_remaining,
		rate_limit_ref, routing_eligible, migration_state, created_at
		FROM deployment_lifecycle_events WHERE deployment_id=$1 ORDER BY effective_at, created_at`, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("list deployment lifecycle events: %w", err)
	}
	defer rows.Close()
	items := make([]domain.DeploymentLifecycleEvent, 0)
	for rows.Next() {
		var event domain.DeploymentLifecycleEvent
		var replacements []byte
		if err := rows.Scan(
			&event.ID, &event.DeploymentID, &event.From, &event.To, &event.AnnouncedAt,
			&event.EffectiveAt, &event.EvidenceRef, &replacements, &event.QuotaRemaining,
			&event.RateLimitRef, &event.RoutingEligible, &event.MigrationState, &event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(replacements, &event.ReplacementIDs); err != nil {
			return nil, err
		}
		items = append(items, event)
	}
	return items, rows.Err()
}
