package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const (
	defaultOutboxDispatchScopeLimit = 100
	maxOutboxDispatchScopeLimit     = 1000
)

// OutboxDispatchScope is global operational scheduling metadata. It contains
// only Tenant/Project identity and timestamps; tenant business payloads remain
// in RLS-protected outbox_messages.
type OutboxDispatchScope struct {
	TenantID    string
	ProjectID   string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

type OutboxDispatchScopeStore interface {
	List(context.Context, int) ([]OutboxDispatchScope, error)
}

type PostgresOutboxDispatchScopes struct {
	db *sql.DB
}

func NewPostgresOutboxDispatchScopes(db *sql.DB) (*PostgresOutboxDispatchScopes, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required for Outbox dispatch scope store")
	}
	return &PostgresOutboxDispatchScopes{db: db}, nil
}

func (s *PostgresOutboxDispatchScopes) List(ctx context.Context, limit int) ([]OutboxDispatchScope, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("Outbox dispatch scope store is not configured")
	}
	if limit <= 0 {
		limit = defaultOutboxDispatchScopeLimit
	}
	if limit > maxOutboxDispatchScopeLimit {
		limit = maxOutboxDispatchScopeLimit
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT tenant_id, project_id, first_seen_at, last_seen_at
FROM outbox_dispatch_scopes
ORDER BY last_seen_at ASC, tenant_id ASC, project_id ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list Outbox dispatch scopes: %w", err)
	}
	defer rows.Close()

	items := make([]OutboxDispatchScope, 0)
	for rows.Next() {
		var item OutboxDispatchScope
		if err := rows.Scan(&item.TenantID, &item.ProjectID, &item.FirstSeenAt, &item.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan Outbox dispatch scope: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Outbox dispatch scopes: %w", err)
	}
	return items, nil
}
