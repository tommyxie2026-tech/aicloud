package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

var (
	ErrAttemptNotFound       = errors.New("execution attempt not found")
	ErrAttemptStateConflict  = errors.New("execution attempt state conflict")
	ErrAttemptBindingInvalid = errors.New("execution attempt binding is invalid")
)

// StartAttempt freezes the target/governance binding and transitions the
// atomically-created PENDING attempt to RUNNING under the current lease fence.
func (r *PostgresExecutionNodeLeases) StartAttempt(ctx context.Context, lease execution.NodeLease, binding execution.AttemptBinding) (execution.AttemptRecord, error) {
	if strings.TrimSpace(string(lease.AttemptRef)) == "" {
		return execution.AttemptRecord{}, fmt.Errorf("%w: lease attempt reference is required", ErrAttemptBindingInvalid)
	}
	if err := validateAttemptBinding(binding); err != nil {
		return execution.AttemptRecord{}, err
	}

	var record execution.AttemptRecord
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		now, err := databaseNow(ctx, tx)
		if err != nil {
			return err
		}
		runtime, err := loadNodeRuntimeForUpdate(ctx, tx, lease.ExecutionRef, lease.PlanRevision, lease.NodeRef)
		if err != nil {
			return err
		}
		if !leaseOwnsRow(runtime, lease, now) {
			return ErrNodeLeaseLost
		}

		result, err := tx.ExecContext(ctx, `UPDATE execution_attempts
			SET status='RUNNING', target_id=$7, target_revision=$8,
				target_snapshot_digest=$9, policy_decision_id=$10,
				budget_reservation_id=$11, started_at=$12, updated_at=$12
			WHERE attempt_id=$1 AND execution_id=$2 AND plan_revision=$3 AND node_id=$4
				AND attempt_number=$5 AND lease_fence=$6 AND status='PENDING'`,
			lease.AttemptRef, lease.ExecutionRef, lease.PlanRevision, lease.NodeRef,
			lease.AttemptNumber, lease.Fence, binding.TargetRef, binding.TargetRevision,
			binding.TargetSnapshot, binding.PolicyDecisionRef, binding.BudgetReservationRef, now)
		if err != nil {
			return fmt.Errorf("start execution attempt: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrAttemptStateConflict
		}
		record, err = loadAttemptTx(ctx, tx, lease.AttemptRef)
		return err
	})
	return record, err
}

func (r *PostgresExecutionNodeLeases) GetAttempt(ctx context.Context, attemptID execution.AttemptID) (execution.AttemptRecord, error) {
	if attemptID == "" {
		return execution.AttemptRecord{}, fmt.Errorf("attempt id is required")
	}
	var record execution.AttemptRecord
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		var err error
		record, err = loadAttemptTx(ctx, tx, attemptID)
		return err
	})
	return record, err
}

func (r *PostgresExecutionNodeLeases) ListAttempts(ctx context.Context, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID) ([]execution.AttemptRecord, error) {
	if executionID == "" || planRevision < 1 || nodeID == "" {
		return nil, fmt.Errorf("execution, positive plan revision and node are required")
	}
	var records []execution.AttemptRecord
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		rows, err := tx.QueryContext(ctx, attemptSelect+`
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
			ORDER BY attempt_number`, executionID, planRevision, nodeID)
		if err != nil {
			return fmt.Errorf("list execution attempts: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			record, err := scanAttempt(rows)
			if err != nil {
				return err
			}
			records = append(records, record)
		}
		return rows.Err()
	})
	return records, err
}

func insertPendingAttemptTx(
	ctx context.Context,
	tx *sql.Tx,
	principal identity.Principal,
	attemptID execution.AttemptID,
	executionID execution.ExecutionID,
	planRevision int64,
	nodeID execution.NodeID,
	attemptNumber int,
	fence int64,
	owner string,
	idempotencyKey string,
	claimedAt time.Time,
) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, idempotency_key,
		claimed_at, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'PENDING',NULLIF($10,''),$11,$11,$11)`,
		principal.TenantID, principal.ProjectID, attemptID, executionID, planRevision,
		nodeID, attemptNumber, fence, owner, idempotencyKey, claimedAt)
	if err != nil {
		return fmt.Errorf("insert pending execution attempt: %w", err)
	}
	return nil
}

func abandonAttemptTx(ctx context.Context, tx *sql.Tx, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID, attemptNumber int, fence int64, finishedAt time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE execution_attempts
		SET status='ABANDONED', finished_at=$6, updated_at=$6
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
			AND attempt_number=$4 AND lease_fence=$5
			AND status IN ('PENDING','RUNNING')`,
		executionID, planRevision, nodeID, attemptNumber, fence, finishedAt)
	if err != nil {
		return fmt.Errorf("abandon execution attempt: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrAttemptStateConflict
	}
	return nil
}

func updateAttemptEffectStartedTx(ctx context.Context, tx *sql.Tx, lease execution.NodeLease, at time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE execution_attempts
		SET effect_started_at=$7, updated_at=$7
		WHERE attempt_id=$1 AND execution_id=$2 AND plan_revision=$3 AND node_id=$4
			AND lease_fence=$5 AND attempt_number=$6 AND status='RUNNING'
			AND effect_started_at IS NULL`,
		lease.AttemptRef, lease.ExecutionRef, lease.PlanRevision, lease.NodeRef,
		lease.Fence, lease.AttemptNumber, at)
	if err != nil {
		return fmt.Errorf("mark attempt effect started: %w", err)
	}
	return requireOneAttemptRow(result)
}

func updateAttemptEffectCommittedTx(ctx context.Context, tx *sql.Tx, lease execution.NodeLease, at time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE execution_attempts
		SET effect_committed_at=$7, updated_at=$7
		WHERE attempt_id=$1 AND execution_id=$2 AND plan_revision=$3 AND node_id=$4
			AND lease_fence=$5 AND attempt_number=$6 AND status='RUNNING'
			AND effect_started_at IS NOT NULL AND effect_committed_at IS NULL`,
		lease.AttemptRef, lease.ExecutionRef, lease.PlanRevision, lease.NodeRef,
		lease.Fence, lease.AttemptNumber, at)
	if err != nil {
		return fmt.Errorf("mark attempt effect committed: %w", err)
	}
	return requireOneAttemptRow(result)
}

func completeAttemptTx(ctx context.Context, tx *sql.Tx, lease execution.NodeLease, completion execution.AttemptCompletion, finishedAt time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE execution_attempts
		SET status=$7, error_class=NULLIF($8,''), input_tokens=$9,
			output_tokens=$10, cost=$11, duration_ms=$12,
			finished_at=$13, updated_at=$13
		WHERE attempt_id=$1 AND execution_id=$2 AND plan_revision=$3 AND node_id=$4
			AND lease_fence=$5 AND attempt_number=$6
			AND (
				status='RUNNING'
				OR ($7='CANCELLED' AND status='PENDING')
			)`,
		lease.AttemptRef, lease.ExecutionRef, lease.PlanRevision, lease.NodeRef,
		lease.Fence, lease.AttemptNumber, completion.Status, completion.ErrorClass,
		completion.Usage.InputTokens, completion.Usage.OutputTokens, completion.Usage.Cost,
		completion.Usage.Duration.Milliseconds(), finishedAt)
	if err != nil {
		return fmt.Errorf("complete execution attempt: %w", err)
	}
	return requireOneAttemptRow(result)
}

func loadCurrentAttemptIDTx(ctx context.Context, tx *sql.Tx, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID, attemptNumber int, fence int64) (execution.AttemptID, error) {
	var attemptID execution.AttemptID
	err := tx.QueryRowContext(ctx, `SELECT attempt_id FROM execution_attempts
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
			AND attempt_number=$4 AND lease_fence=$5`,
		executionID, planRevision, nodeID, attemptNumber, fence).Scan(&attemptID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrAttemptNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load current execution attempt id: %w", err)
	}
	return attemptID, nil
}

const attemptSelect = `SELECT attempt_id, execution_id, plan_revision, node_id,
	attempt_number, lease_fence, lease_owner, status,
	target_id, target_revision, target_snapshot_digest,
	policy_decision_id, budget_reservation_id, COALESCE(idempotency_key,''),
	COALESCE(error_class,''), input_tokens, output_tokens, cost, duration_ms,
	claimed_at, started_at, effect_started_at, effect_committed_at,
	finished_at, created_at, updated_at
	FROM execution_attempts`

func loadAttemptTx(ctx context.Context, tx *sql.Tx, attemptID execution.AttemptID) (execution.AttemptRecord, error) {
	row := tx.QueryRowContext(ctx, attemptSelect+` WHERE attempt_id=$1`, attemptID)
	record, err := scanAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return execution.AttemptRecord{}, ErrAttemptNotFound
	}
	return record, err
}

type attemptScanner interface {
	Scan(dest ...any) error
}

func scanAttempt(scanner attemptScanner) (execution.AttemptRecord, error) {
	var record execution.AttemptRecord
	var targetID sql.NullString
	var targetRevision sql.NullInt64
	var targetSnapshot sql.NullString
	var policyRef sql.NullString
	var budgetRef sql.NullString
	var errorClass string
	var cost float64
	var durationMS int64
	var startedAt, effectStartedAt, effectCommittedAt, finishedAt sql.NullTime

	err := scanner.Scan(
		&record.Attempt.ID, &record.Attempt.ExecutionRef, &record.PlanRevision,
		&record.Attempt.NodeRef, &record.Attempt.AttemptNumber, &record.LeaseFence,
		&record.LeaseOwner, &record.Attempt.Status, &targetID, &targetRevision,
		&targetSnapshot, &policyRef, &budgetRef, &record.Attempt.IdempotencyKey,
		&errorClass, &record.Attempt.Usage.InputTokens, &record.Attempt.Usage.OutputTokens,
		&cost, &durationMS, &record.ClaimedAt, &startedAt, &effectStartedAt,
		&effectCommittedAt, &finishedAt, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return execution.AttemptRecord{}, err
	}
	if targetID.Valid {
		record.Attempt.TargetRef = execution.TargetID(targetID.String)
	}
	if targetRevision.Valid {
		record.Attempt.TargetRevision = targetRevision.Int64
	}
	if targetSnapshot.Valid {
		record.Attempt.TargetSnapshot = targetSnapshot.String
	}
	if policyRef.Valid {
		record.PolicyDecisionRef = execution.PolicyDecisionID(policyRef.String)
	}
	if budgetRef.Valid {
		record.BudgetReservationRef = budgetRef.String
	}
	if errorClass != "" {
		record.Attempt.ErrorClass = execution.ErrorClass(errorClass)
	}
	record.Attempt.Usage.Cost = cost
	record.Attempt.Usage.Duration = time.Duration(durationMS) * time.Millisecond
	if startedAt.Valid {
		record.Attempt.StartedAt = startedAt.Time
	}
	if effectStartedAt.Valid {
		v := effectStartedAt.Time
		record.Attempt.EffectStartedAt = &v
	}
	if effectCommittedAt.Valid {
		v := effectCommittedAt.Time
		record.Attempt.EffectCommittedAt = &v
	}
	if finishedAt.Valid {
		v := finishedAt.Time
		record.Attempt.FinishedAt = &v
	}
	return record, nil
}

func validateAttemptBinding(binding execution.AttemptBinding) error {
	if binding.TargetRef == "" || binding.TargetRevision < 1 || strings.TrimSpace(binding.TargetSnapshot) == "" {
		return fmt.Errorf("%w: target id, positive revision and snapshot are required", ErrAttemptBindingInvalid)
	}
	if binding.PolicyDecisionRef == "" {
		return fmt.Errorf("%w: policy decision is required", ErrAttemptBindingInvalid)
	}
	if strings.TrimSpace(binding.BudgetReservationRef) == "" {
		return fmt.Errorf("%w: budget reservation is required", ErrAttemptBindingInvalid)
	}
	return nil
}

func requireOneAttemptRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrAttemptStateConflict
	}
	return nil
}

func newAttemptID() (execution.AttemptID, error) {
	token, err := newLeaseToken()
	if err != nil {
		return "", err
	}
	return execution.AttemptID("att_" + token), nil
}
