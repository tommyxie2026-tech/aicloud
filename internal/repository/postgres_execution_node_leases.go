package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

var (
	ErrNodeRuntimeNotFound = errors.New("execution node runtime not found")
	ErrNodeRuntimeExists   = errors.New("execution node runtime already exists")
	ErrNodeNotClaimable    = errors.New("execution node is not claimable")
	ErrNodeLeaseHeld       = errors.New("execution node lease is held by another worker")
	ErrNodeLeaseLost       = errors.New("execution node lease is no longer valid")
	ErrNodeRecoveryRequired = errors.New("execution node requires recovery review before retry")
	ErrNodeEffectStarted   = errors.New("execution node side effect has already started")
)

type PostgresExecutionNodeLeases struct {
	db *sql.DB
}

func NewPostgresExecutionNodeLeases(db *sql.DB) *PostgresExecutionNodeLeases {
	return &PostgresExecutionNodeLeases{db: db}
}

// Register creates the durable runtime row for a node in an immutable plan
// revision. Plan-derived effect and retry semantics are frozen at registration.
func (r *PostgresExecutionNodeLeases) Register(ctx context.Context, record execution.NodeRuntimeRecord) error {
	if err := validateNodeRuntimeRecord(record); err != nil {
		return err
	}
	return r.withScopedTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		result, err := tx.ExecContext(ctx, `INSERT INTO execution_node_runtime(
			tenant_id, project_id, execution_id, plan_revision, node_id, state,
			effect_class, retry_safe, idempotency_key, attempt_number, lease_fence,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,''),$10,$11,NOW(),NOW())
		ON CONFLICT (tenant_id, project_id, execution_id, plan_revision, node_id) DO NOTHING`,
			principal.TenantID, principal.ProjectID, record.ExecutionRef, record.PlanRevision,
			record.NodeRef, record.State, record.EffectClass, record.RetrySafe,
			record.IdempotencyKey, record.AttemptNumber, record.LeaseFence)
		if err != nil {
			return fmt.Errorf("register execution node runtime: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeRuntimeExists
		}
		return nil
	})
}

// Claim atomically authorizes one worker to execute the next attempt. READY
// nodes may be claimed immediately. Expired RUNNING nodes may be reclaimed only
// when the frozen retry semantics say retry_safe and there is no evidence that
// the previous attempt committed its effect.
func (r *PostgresExecutionNodeLeases) Claim(
	ctx context.Context,
	executionID execution.ExecutionID,
	planRevision int64,
	nodeID execution.NodeID,
	owner string,
	now time.Time,
	leaseDuration time.Duration,
) (execution.NodeLease, error) {
	if err := validateLeaseIdentity(executionID, planRevision, nodeID, owner, now, leaseDuration); err != nil {
		return execution.NodeLease{}, err
	}

	var lease execution.NodeLease
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		row, err := loadNodeRuntimeForUpdate(ctx, tx, executionID, planRevision, nodeID)
		if err != nil {
			return err
		}
		if row.record.Terminal() {
			return ErrNodeNotClaimable
		}

		switch row.record.State {
		case execution.NodeReady:
			if row.leaseToken.Valid {
				return ErrNodeLeaseHeld
			}
		case execution.NodeRunning:
			if row.leaseExpiresAt.Valid && now.Before(row.leaseExpiresAt.Time) {
				return ErrNodeLeaseHeld
			}
			if !row.record.RetrySafe || row.record.EffectCommittedAt != nil {
				return ErrNodeRecoveryRequired
			}
		default:
			return ErrNodeNotClaimable
		}

		token, err := newLeaseToken()
		if err != nil {
			return err
		}
		expiresAt := now.Add(leaseDuration)
		attempt := row.record.AttemptNumber + 1
		fence := row.record.LeaseFence + 1

		result, err := tx.ExecContext(ctx, `UPDATE execution_node_runtime
			SET state='RUNNING', attempt_number=$4, lease_owner=$5, lease_token=$6,
				lease_fence=$7, claimed_at=$8, heartbeat_at=$8, lease_expires_at=$9,
				effect_started_at=NULL, effect_committed_at=NULL, updated_at=$8
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3`,
			executionID, planRevision, nodeID, attempt, owner, token, fence, now, expiresAt)
		if err != nil {
			return fmt.Errorf("claim execution node: %w", err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return err
		} else if affected != 1 {
			return ErrNodeLeaseLost
		}

		lease = execution.NodeLease{
			ExecutionRef: executionID, PlanRevision: planRevision, NodeRef: nodeID,
			OwnerWorkerID: owner, Token: token, Fence: fence, AttemptNumber: attempt,
			ClaimedAt: now, HeartbeatAt: now, ExpiresAt: expiresAt,
		}
		return nil
	})
	return lease, err
}

// Renew extends a currently active lease. An expired lease cannot be revived;
// the node must go through Claim/recovery so a new fence is issued.
func (r *PostgresExecutionNodeLeases) Renew(ctx context.Context, lease execution.NodeLease, now time.Time, leaseDuration time.Duration) (execution.NodeLease, error) {
	if err := validateHeldLease(lease, now, leaseDuration); err != nil {
		return execution.NodeLease{}, err
	}
	expiresAt := now.Add(leaseDuration)
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		result, err := tx.ExecContext(ctx, `UPDATE execution_node_runtime
			SET heartbeat_at=$7, lease_expires_at=$8, updated_at=$7
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
				AND state='RUNNING' AND lease_owner=$4 AND lease_token=$5 AND lease_fence=$6
				AND lease_expires_at > $7`,
			lease.ExecutionRef, lease.PlanRevision, lease.NodeRef, lease.OwnerWorkerID,
			lease.Token, lease.Fence, now, expiresAt)
		if err != nil {
			return fmt.Errorf("renew execution node lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeLeaseLost
		}
		return nil
	})
	if err != nil {
		return execution.NodeLease{}, err
	}
	lease.HeartbeatAt = now
	lease.ExpiresAt = expiresAt
	return lease, nil
}

func (r *PostgresExecutionNodeLeases) MarkEffectStarted(ctx context.Context, lease execution.NodeLease, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("effect start time is required")
	}
	return r.fencedUpdate(ctx, lease, at, `
		UPDATE execution_node_runtime
		SET effect_started_at=$7, updated_at=$7
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
			AND state='RUNNING' AND lease_owner=$4 AND lease_token=$5 AND lease_fence=$6
			AND lease_expires_at > $7 AND effect_started_at IS NULL`)
}

func (r *PostgresExecutionNodeLeases) MarkEffectCommitted(ctx context.Context, lease execution.NodeLease, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("effect commit time is required")
	}
	return r.fencedUpdate(ctx, lease, at, `
		UPDATE execution_node_runtime
		SET effect_committed_at=$7, updated_at=$7
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
			AND state='RUNNING' AND lease_owner=$4 AND lease_token=$5 AND lease_fence=$6
			AND lease_expires_at > $7 AND effect_started_at IS NOT NULL
			AND effect_committed_at IS NULL`)
}

// ReleaseBeforeEffect returns a claimed node to READY only if no side effect was
// marked as started. Once an effect starts, retry/recovery must use the explicit
// failure and idempotency semantics instead of silently releasing the claim.
func (r *PostgresExecutionNodeLeases) ReleaseBeforeEffect(ctx context.Context, lease execution.NodeLease, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("release time is required")
	}
	return r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		result, err := tx.ExecContext(ctx, `UPDATE execution_node_runtime
			SET state='READY', lease_owner=NULL, lease_token=NULL, claimed_at=NULL,
				heartbeat_at=NULL, lease_expires_at=NULL, updated_at=$7
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
				AND state='RUNNING' AND lease_owner=$4 AND lease_token=$5 AND lease_fence=$6
				AND effect_started_at IS NULL`,
			lease.ExecutionRef, lease.PlanRevision, lease.NodeRef, lease.OwnerWorkerID,
			lease.Token, lease.Fence, at)
		if err != nil {
			return fmt.Errorf("release execution node lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeLeaseLost
		}
		return nil
	})
}

// Complete transitions the node to a terminal state under the current active
// fence and clears the lease. A worker cannot complete after lease expiry.
func (r *PostgresExecutionNodeLeases) Complete(ctx context.Context, lease execution.NodeLease, terminal execution.NodeState, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("completion time is required")
	}
	if !((execution.NodeRuntimeRecord{State: terminal}).Terminal()) {
		return fmt.Errorf("terminal node state is required")
	}
	return r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		result, err := tx.ExecContext(ctx, `UPDATE execution_node_runtime
			SET state=$7, lease_owner=NULL, lease_token=NULL, claimed_at=NULL,
				heartbeat_at=NULL, lease_expires_at=NULL, updated_at=$8
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3
				AND state='RUNNING' AND lease_owner=$4 AND lease_token=$5 AND lease_fence=$6
				AND lease_expires_at > $8`,
			lease.ExecutionRef, lease.PlanRevision, lease.NodeRef, lease.OwnerWorkerID,
			lease.Token, lease.Fence, terminal, at)
		if err != nil {
			return fmt.Errorf("complete execution node: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeLeaseLost
		}
		return nil
	})
}

func (r *PostgresExecutionNodeLeases) Get(ctx context.Context, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID) (execution.NodeRuntimeRecord, *execution.NodeLease, error) {
	var record execution.NodeRuntimeRecord
	var lease *execution.NodeLease
	err := r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		row, err := loadNodeRuntime(ctx, tx, executionID, planRevision, nodeID, false)
		if err != nil {
			return err
		}
		record = row.record
		if row.leaseToken.Valid {
			lease = &execution.NodeLease{
				ExecutionRef: executionID, PlanRevision: planRevision, NodeRef: nodeID,
				OwnerWorkerID: row.leaseOwner.String, Token: row.leaseToken.String,
				Fence: record.LeaseFence, AttemptNumber: record.AttemptNumber,
				ClaimedAt: row.claimedAt.Time, HeartbeatAt: row.heartbeatAt.Time,
				ExpiresAt: row.leaseExpiresAt.Time,
			}
		}
		return nil
	})
	return record, lease, err
}

func (r *PostgresExecutionNodeLeases) fencedUpdate(ctx context.Context, lease execution.NodeLease, at time.Time, query string) error {
	return r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		result, err := tx.ExecContext(ctx, query,
			lease.ExecutionRef, lease.PlanRevision, lease.NodeRef, lease.OwnerWorkerID,
			lease.Token, lease.Fence, at)
		if err != nil {
			return fmt.Errorf("fenced execution node update: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeLeaseLost
		}
		return nil
	})
}

type nodeRuntimeRow struct {
	record         execution.NodeRuntimeRecord
	leaseOwner     sql.NullString
	leaseToken     sql.NullString
	claimedAt      sql.NullTime
	heartbeatAt    sql.NullTime
	leaseExpiresAt sql.NullTime
}

func loadNodeRuntimeForUpdate(ctx context.Context, tx *sql.Tx, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID) (nodeRuntimeRow, error) {
	return loadNodeRuntime(ctx, tx, executionID, planRevision, nodeID, true)
}

func loadNodeRuntime(ctx context.Context, tx *sql.Tx, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID, forUpdate bool) (nodeRuntimeRow, error) {
	query := `SELECT execution_id, plan_revision, node_id, state, effect_class, retry_safe,
		COALESCE(idempotency_key,''), attempt_number, lease_owner, lease_token, lease_fence,
		claimed_at, heartbeat_at, lease_expires_at, effect_started_at, effect_committed_at, updated_at
		FROM execution_node_runtime
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var row nodeRuntimeRow
	var effectStarted, effectCommitted sql.NullTime
	err := tx.QueryRowContext(ctx, query, executionID, planRevision, nodeID).Scan(
		&row.record.ExecutionRef, &row.record.PlanRevision, &row.record.NodeRef,
		&row.record.State, &row.record.EffectClass, &row.record.RetrySafe,
		&row.record.IdempotencyKey, &row.record.AttemptNumber, &row.leaseOwner,
		&row.leaseToken, &row.record.LeaseFence, &row.claimedAt, &row.heartbeatAt,
		&row.leaseExpiresAt, &effectStarted, &effectCommitted, &row.record.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nodeRuntimeRow{}, ErrNodeRuntimeNotFound
	}
	if err != nil {
		return nodeRuntimeRow{}, fmt.Errorf("load execution node runtime: %w", err)
	}
	if effectStarted.Valid {
		v := effectStarted.Time
		row.record.EffectStartedAt = &v
	}
	if effectCommitted.Valid {
		v := effectCommitted.Time
		row.record.EffectCommittedAt = &v
	}
	return row, nil
}

func (r *PostgresExecutionNodeLeases) withScopedTx(ctx context.Context, fn func(*sql.Tx, identity.Principal) error) error {
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
		return fmt.Errorf("begin execution node transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return fmt.Errorf("set execution node transaction scope: %w", err)
	}
	if err := fn(tx, principal); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit execution node transaction: %w", err)
	}
	return nil
}

func validateNodeRuntimeRecord(record execution.NodeRuntimeRecord) error {
	if record.ExecutionRef == "" || record.PlanRevision < 1 || record.NodeRef == "" {
		return fmt.Errorf("execution, positive plan revision and node are required")
	}
	if record.State == "" {
		return fmt.Errorf("node state is required")
	}
	if record.EffectClass == execution.EffectIdempotentMutation && strings.TrimSpace(record.IdempotencyKey) == "" {
		return fmt.Errorf("idempotent mutation requires idempotency key")
	}
	if record.EffectClass == execution.EffectNonIdempotentMutation && record.RetrySafe {
		return fmt.Errorf("non-idempotent mutation cannot be registered retry-safe")
	}
	return nil
}

func validateLeaseIdentity(executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID, owner string, now time.Time, leaseDuration time.Duration) error {
	if executionID == "" || planRevision < 1 || nodeID == "" || strings.TrimSpace(owner) == "" {
		return fmt.Errorf("execution, positive plan revision, node and lease owner are required")
	}
	if now.IsZero() || leaseDuration <= 0 {
		return fmt.Errorf("current time and positive lease duration are required")
	}
	return nil
}

func validateHeldLease(lease execution.NodeLease, now time.Time, leaseDuration time.Duration) error {
	if err := validateLeaseIdentity(lease.ExecutionRef, lease.PlanRevision, lease.NodeRef, lease.OwnerWorkerID, now, leaseDuration); err != nil {
		return err
	}
	if strings.TrimSpace(lease.Token) == "" || lease.Fence < 1 {
		return fmt.Errorf("lease token and positive fence are required")
	}
	return nil
}

func newLeaseToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate lease token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
