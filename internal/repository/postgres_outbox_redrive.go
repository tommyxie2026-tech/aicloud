package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
)

var ErrOutboxNotDeadLetter = errors.New("outbox message is not dead letter")

type OutboxRedriveEvent struct {
	RedriveEventID    string
	TenantID          string
	ProjectID         string
	OutboxID          string
	ActorPrincipalType string
	ActorSubjectID    string
	Reason            string
	PreviousAttempts  int
	PreviousLastError string
	RedrivenAt        time.Time
}

// RedriveDeadLetter explicitly returns one dead-letter Outbox message to the
// normal pending delivery path. It never calls a downstream adapter directly.
// The status change and append-only redrive evidence are committed atomically
// under the current Tenant/Project RLS scope.
func (r *ScopedPostgresOutbox) RedriveDeadLetter(
	ctx context.Context,
	outboxID string,
	reason string,
	redrivenAt time.Time,
	availableAt time.Time,
) (OutboxRedriveEvent, error) {
	if r == nil || r.db == nil {
		return OutboxRedriveEvent{}, fmt.Errorf("database is required")
	}
	outboxID = strings.TrimSpace(outboxID)
	reason = strings.TrimSpace(reason)
	if outboxID == "" || reason == "" || redrivenAt.IsZero() || availableAt.IsZero() {
		return OutboxRedriveEvent{}, fmt.Errorf("outbox id, reason, redrive time and available time are required")
	}
	if availableAt.Before(redrivenAt) {
		return OutboxRedriveEvent{}, fmt.Errorf("Outbox redrive available time cannot be before redrive time")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return OutboxRedriveEvent{}, err
	}

	event := OutboxRedriveEvent{
		RedriveEventID:     tracepkg.NewID("outbox-redrive"),
		TenantID:           principal.TenantID,
		ProjectID:          principal.ProjectID,
		OutboxID:           outboxID,
		ActorPrincipalType: string(principal.Type),
		ActorSubjectID:     principal.SubjectID,
		Reason:             reason,
		RedrivenAt:         redrivenAt.UTC(),
	}

	err = r.withScopedTx(ctx, func(tx *sql.Tx) error {
		var status domain.OutboxStatus
		if err := tx.QueryRowContext(ctx, `
SELECT status, attempts, COALESCE(last_error, '')
FROM outbox_messages
WHERE outbox_id=$1 AND tenant_id=$2 AND project_id=$3
FOR UPDATE`, outboxID, principal.TenantID, principal.ProjectID).Scan(
			&status,
			&event.PreviousAttempts,
			&event.PreviousLastError,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("load dead-letter Outbox message: %w", err)
		}
		if status != domain.OutboxDeadLetter {
			return ErrOutboxNotDeadLetter
		}

		result, err := tx.ExecContext(ctx, `
UPDATE outbox_messages
SET status='pending',
    available_at=$4,
    lease_owner=NULL,
    lease_expires_at=NULL,
    delivered_at=NULL
WHERE outbox_id=$1
  AND tenant_id=$2
  AND project_id=$3
  AND status='dead_letter'`,
			outboxID, principal.TenantID, principal.ProjectID, availableAt.UTC())
		if err != nil {
			return fmt.Errorf("redrive dead-letter Outbox message: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrOutboxNotDeadLetter
		}

		if _, err := tx.ExecContext(ctx, `
INSERT INTO outbox_redrive_events(
    redrive_event_id, tenant_id, project_id, outbox_id,
    actor_principal_type, actor_subject_id, reason,
    previous_attempts, previous_last_error, redriven_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			event.RedriveEventID,
			event.TenantID,
			event.ProjectID,
			event.OutboxID,
			event.ActorPrincipalType,
			event.ActorSubjectID,
			event.Reason,
			event.PreviousAttempts,
			event.PreviousLastError,
			event.RedrivenAt,
		); err != nil {
			return fmt.Errorf("append Outbox redrive evidence: %w", err)
		}
		return nil
	})
	if err != nil {
		return OutboxRedriveEvent{}, err
	}
	return event, nil
}
