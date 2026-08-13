package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

var ErrOutboxLeaseLost = errors.New("outbox delivery lease is not owned by dispatcher")

type LeasedOutboxMessage struct {
	Message        domain.OutboxMessage
	LeaseOwner     string
	LeaseExpiresAt time.Time
	LastError      string
}

// ScopedPostgresOutbox provides project-scoped dispatcher primitives. It does
// not bypass RLS: callers must process an explicit tenant/project scope.
type ScopedPostgresOutbox struct {
	db *sql.DB
}

func NewScopedPostgresOutbox(db *sql.DB) *ScopedPostgresOutbox {
	return &ScopedPostgresOutbox{db: db}
}

func (r *ScopedPostgresOutbox) Lease(ctx context.Context, owner string, limit int, now time.Time, leaseDuration time.Duration) ([]LeasedOutboxMessage, error) {
	owner = strings.TrimSpace(owner)
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if owner == "" || limit < 1 || leaseDuration <= 0 || now.IsZero() {
		return nil, fmt.Errorf("lease owner, positive limit, current time and lease duration are required")
	}
	if limit > 1000 {
		limit = 1000
	}
	leaseExpiresAt := now.Add(leaseDuration)
	var leased []LeasedOutboxMessage
	err := r.withScopedTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			WITH candidates AS (
				SELECT outbox_id
				FROM outbox_messages
				WHERE (
					(status = 'pending' AND available_at <= $1)
					OR
					(status = 'delivering' AND lease_expires_at <= $1)
				)
				ORDER BY available_at, created_at, outbox_id
				FOR UPDATE SKIP LOCKED
				LIMIT $2
			)
			UPDATE outbox_messages AS o
			SET status='delivering',
				lease_owner=$3,
				lease_expires_at=$4,
				attempts=o.attempts+1
			FROM candidates AS c
			WHERE o.outbox_id=c.outbox_id
			RETURNING o.outbox_id, o.tenant_id, o.project_id, COALESCE(o.task_id,''),
				o.aggregate_type, o.aggregate_id, o.event_type, o.payload::text,
				o.destination, o.idempotency_key, o.status, o.attempts,
				o.available_at, o.created_at, o.delivered_at,
				o.lease_owner, o.lease_expires_at, COALESCE(o.last_error,'')`,
			now, limit, owner, leaseExpiresAt)
		if err != nil {
			return fmt.Errorf("lease outbox messages: %w", err)
		}
		defer rows.Close()
		leased = make([]LeasedOutboxMessage, 0)
		for rows.Next() {
			item, err := scanLeasedOutbox(rows)
			if err != nil {
				return err
			}
			leased = append(leased, item)
		}
		return rows.Err()
	})
	return leased, err
}

func (r *ScopedPostgresOutbox) MarkDelivered(ctx context.Context, outboxID, owner string, deliveredAt time.Time) error {
	if strings.TrimSpace(outboxID) == "" || strings.TrimSpace(owner) == "" || deliveredAt.IsZero() {
		return fmt.Errorf("outbox id, lease owner and delivered time are required")
	}
	return r.withScopedTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE outbox_messages
			SET status='delivered', delivered_at=$3, lease_owner=NULL,
				lease_expires_at=NULL, last_error=NULL
			WHERE outbox_id=$1 AND status='delivering' AND lease_owner=$2`,
			outboxID, owner, deliveredAt)
		if err != nil {
			return fmt.Errorf("mark outbox delivered: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrOutboxLeaseLost
		}
		return nil
	})
}

// FailDelivery returns a leased message to pending with a caller-computed
// backoff time, or moves it to dead_letter once maxAttempts has been reached.
func (r *ScopedPostgresOutbox) FailDelivery(ctx context.Context, outboxID, owner string, now, nextAvailableAt time.Time, maxAttempts int, reason string) (domain.OutboxStatus, error) {
	if strings.TrimSpace(outboxID) == "" || strings.TrimSpace(owner) == "" || now.IsZero() || nextAvailableAt.IsZero() || maxAttempts < 1 {
		return "", fmt.Errorf("outbox id, lease owner, times and positive max attempts are required")
	}
	var finalStatus domain.OutboxStatus
	err := r.withScopedTx(ctx, func(tx *sql.Tx) error {
		var attempts int
		if err := tx.QueryRowContext(ctx, `SELECT attempts FROM outbox_messages
			WHERE outbox_id=$1 AND status='delivering' AND lease_owner=$2 FOR UPDATE`, outboxID, owner).Scan(&attempts); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrOutboxLeaseLost
			}
			return fmt.Errorf("load leased outbox message: %w", err)
		}
		finalStatus = domain.OutboxPending
		availableAt := nextAvailableAt
		if attempts >= maxAttempts {
			finalStatus = domain.OutboxDeadLetter
			availableAt = now
		}
		result, err := tx.ExecContext(ctx, `UPDATE outbox_messages
			SET status=$3, available_at=$4, lease_owner=NULL, lease_expires_at=NULL,
				last_error=NULLIF($5,'')
			WHERE outbox_id=$1 AND status='delivering' AND lease_owner=$2`,
			outboxID, owner, finalStatus, availableAt, strings.TrimSpace(reason))
		if err != nil {
			return fmt.Errorf("record outbox delivery failure: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrOutboxLeaseLost
		}
		return nil
	})
	return finalStatus, err
}

func (r *ScopedPostgresOutbox) withScopedTx(ctx context.Context, fn func(*sql.Tx) error) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("database is required")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scoped outbox transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return fmt.Errorf("set outbox transaction scope: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scoped outbox transaction: %w", err)
	}
	return nil
}

func scanLeasedOutbox(row scanner) (LeasedOutboxMessage, error) {
	var item LeasedOutboxMessage
	var payload []byte
	if err := row.Scan(
		&item.Message.OutboxID, &item.Message.TenantID, &item.Message.ProjectID,
		&item.Message.TaskID, &item.Message.AggregateType, &item.Message.AggregateID,
		&item.Message.EventType, &payload, &item.Message.Destination,
		&item.Message.IdempotencyKey, &item.Message.Status, &item.Message.Attempts,
		&item.Message.AvailableAt, &item.Message.CreatedAt, &item.Message.DeliveredAt,
		&item.LeaseOwner, &item.LeaseExpiresAt, &item.LastError,
	); err != nil {
		return LeasedOutboxMessage{}, fmt.Errorf("scan leased outbox message: %w", err)
	}
	if !json.Valid(payload) {
		return LeasedOutboxMessage{}, fmt.Errorf("outbox %q contains invalid JSON payload", item.Message.OutboxID)
	}
	item.Message.Payload = append(item.Message.Payload[:0], payload...)
	return item, nil
}
