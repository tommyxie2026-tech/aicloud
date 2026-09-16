package repository

import (
	"context"
	"database/sql"
	"fmt"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

// RequeueFailed transitions one failed retry-safe read-only node back to READY.
// Mutation nodes are deliberately excluded because a technical failure does not
// prove that an external side effect was not applied.
func (r *PostgresExecutionNodeLeases) RequeueFailed(ctx context.Context, executionID execution.ExecutionID, planRevision int64, nodeID execution.NodeID) error {
	if executionID == "" || planRevision < 1 || nodeID == "" {
		return fmt.Errorf("execution, positive plan revision and node are required")
	}
	return r.withScopedTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		now, err := databaseNow(ctx, tx)
		if err != nil {
			return err
		}
		row, err := loadNodeRuntimeForUpdate(ctx, tx, executionID, planRevision, nodeID)
		if err != nil {
			return err
		}
		if row.record.State != execution.NodeFailed {
			return ErrNodeNotClaimable
		}
		if !row.record.RetrySafe || isMutation(row.record.EffectClass) {
			return ErrNodeRecoveryRequired
		}
		result, err := tx.ExecContext(ctx, `UPDATE execution_node_runtime
			SET state='READY', lease_owner=NULL, lease_token=NULL, claimed_at=NULL,
				heartbeat_at=NULL, lease_expires_at=NULL, updated_at=$4
			WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3 AND state='FAILED'`,
			executionID, planRevision, nodeID, now)
		if err != nil {
			return fmt.Errorf("requeue failed execution node: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrNodeNotClaimable
		}
		return nil
	})
}
