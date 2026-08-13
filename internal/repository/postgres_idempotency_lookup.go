package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

// ResolveIdempotency lets the Control Plane replay a completed public command
// before recomputing volatile business work such as routing. A concurrent
// absent lookup is only advisory; the write transaction still reserves the key
// and remains the final concurrency authority.
func (r *ScopedPostgresTaskCommands) ResolveIdempotency(ctx context.Context, lookup IdempotencyLookup) (domain.IdempotencyRecord, bool, error) {
	if r == nil || r.db == nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("database is required")
	}
	if strings.TrimSpace(lookup.Operation) == "" || strings.TrimSpace(lookup.Key) == "" || strings.TrimSpace(lookup.RequestDigest) == "" {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("operation, idempotency key and request digest are required")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if lookup.TenantID != principal.TenantID || lookup.ProjectID != principal.ProjectID {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("idempotency lookup scope must match authenticated principal")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("begin idempotency lookup transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("set idempotency lookup scope: %w", err)
	}

	key := domain.IdempotencyRecord{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		Operation: lookup.Operation, Key: lookup.Key,
	}
	record, err := loadIdempotency(ctx, tx, key)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return domain.IdempotencyRecord{}, false, fmt.Errorf("commit empty idempotency lookup: %w", err)
		}
		return domain.IdempotencyRecord{}, false, nil
	}
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	if record.RequestDigest != lookup.RequestDigest {
		return domain.IdempotencyRecord{}, false, ErrIdempotencyConflict
	}
	if err := tx.Commit(); err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("commit idempotency lookup: %w", err)
	}
	switch record.Status {
	case domain.IdempotencyCompleted, domain.IdempotencyFailedFinal:
		return record, true, nil
	case domain.IdempotencyInProgress:
		return domain.IdempotencyRecord{}, false, ErrIdempotencyInProgress
	case domain.IdempotencyFailedRetryable:
		return record, false, nil
	default:
		return domain.IdempotencyRecord{}, false, fmt.Errorf("unsupported idempotency status %q", record.Status)
	}
}

func loadIdempotency(ctx context.Context, tx *sql.Tx, key domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	var responsePayload []byte
	err := tx.QueryRowContext(ctx, `SELECT tenant_id, project_id, operation,
		idempotency_key, request_digest, status, COALESCE(resource_id,''),
		COALESCE(response_code,0), COALESCE(response_digest,''),
		COALESCE(response_payload,'null'::jsonb)::text, created_at, expires_at
		FROM idempotency_records
		WHERE tenant_id=$1 AND project_id=$2 AND operation=$3 AND idempotency_key=$4`,
		key.TenantID, key.ProjectID, key.Operation, key.Key).Scan(
		&record.TenantID, &record.ProjectID, &record.Operation, &record.Key,
		&record.RequestDigest, &record.Status, &record.ResourceID,
		&record.ResponseCode, &record.ResponseDigest, &responsePayload,
		&record.CreatedAt, &record.ExpiresAt,
	)
	if err != nil {
		return domain.IdempotencyRecord{}, err
	}
	if string(responsePayload) != "null" {
		record.ResponsePayload = append(record.ResponsePayload[:0], responsePayload...)
	}
	return record, nil
}
