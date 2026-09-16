package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

// ScopedPostgresExecutionLineage persists Goal -> PlanRevision -> Execution
// parent records under the same tenant/project RLS boundary as canonical Tasks.
type ScopedPostgresExecutionLineage struct {
	db *sql.DB
}

func NewScopedPostgresExecutionLineage(db *sql.DB) *ScopedPostgresExecutionLineage {
	return &ScopedPostgresExecutionLineage{db: db}
}

var _ ExecutionLineageStore = (*ScopedPostgresExecutionLineage)(nil)

func (r *ScopedPostgresExecutionLineage) CreateGoal(ctx context.Context, goal execution.Goal) (execution.Goal, error) {
	if err := validateGoalLineage(goal); err != nil {
		return execution.Goal{}, err
	}
	criteria, err := json.Marshal(goal.AcceptanceCriteria)
	if err != nil {
		return execution.Goal{}, fmt.Errorf("encode goal acceptance criteria: %w", err)
	}
	constraints, err := json.Marshal(goal.Constraints)
	if err != nil {
		return execution.Goal{}, fmt.Errorf("encode goal constraints: %w", err)
	}
	budget, err := json.Marshal(goal.Budget)
	if err != nil {
		return execution.Goal{}, fmt.Errorf("encode goal budget: %w", err)
	}

	err = r.withProjectTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		if err := requireScopedTask(ctx, tx, goal.TaskRef); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO execution_goals (
			tenant_id, project_id, goal_id, task_id, objective,
			acceptance_criteria, constraints, budget
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			principal.TenantID, principal.ProjectID, string(goal.ID), goal.TaskRef,
			goal.Objective, criteria, constraints, budget)
		if err != nil {
			return fmt.Errorf("create execution goal: %w", err)
		}
		return nil
	})
	if err != nil {
		return execution.Goal{}, err
	}
	return goal, nil
}

func (r *ScopedPostgresExecutionLineage) GetGoal(ctx context.Context, goalID execution.GoalID) (execution.Goal, error) {
	var goal execution.Goal
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		var criteria, constraints, budget []byte
		err := tx.QueryRowContext(ctx, `SELECT goal_id, task_id, objective,
			acceptance_criteria, constraints, budget
			FROM execution_goals WHERE goal_id=$1`, string(goalID)).Scan(
			&goal.ID, &goal.TaskRef, &goal.Objective, &criteria, &constraints, &budget,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get execution goal: %w", err)
		}
		if err := json.Unmarshal(criteria, &goal.AcceptanceCriteria); err != nil {
			return fmt.Errorf("decode goal acceptance criteria: %w", err)
		}
		if err := json.Unmarshal(constraints, &goal.Constraints); err != nil {
			return fmt.Errorf("decode goal constraints: %w", err)
		}
		if err := json.Unmarshal(budget, &goal.Budget); err != nil {
			return fmt.Errorf("decode goal budget: %w", err)
		}
		return nil
	})
	return goal, err
}

func (r *ScopedPostgresExecutionLineage) ListGoalsByTask(ctx context.Context, taskID string) ([]execution.Goal, error) {
	items := make([]execution.Goal, 0)
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		rows, err := tx.QueryContext(ctx, `SELECT goal_id, task_id, objective,
			acceptance_criteria, constraints, budget
			FROM execution_goals WHERE task_id=$1 ORDER BY created_at, goal_id`, taskID)
		if err != nil {
			return fmt.Errorf("list execution goals: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var goal execution.Goal
			var criteria, constraints, budget []byte
			if err := rows.Scan(&goal.ID, &goal.TaskRef, &goal.Objective, &criteria, &constraints, &budget); err != nil {
				return fmt.Errorf("scan execution goal: %w", err)
			}
			if err := json.Unmarshal(criteria, &goal.AcceptanceCriteria); err != nil {
				return fmt.Errorf("decode goal acceptance criteria: %w", err)
			}
			if err := json.Unmarshal(constraints, &goal.Constraints); err != nil {
				return fmt.Errorf("decode goal constraints: %w", err)
			}
			if err := json.Unmarshal(budget, &goal.Budget); err != nil {
				return fmt.Errorf("decode goal budget: %w", err)
			}
			items = append(items, goal)
		}
		return rows.Err()
	})
	return items, err
}

func (r *ScopedPostgresExecutionLineage) CreatePlan(ctx context.Context, plan execution.ExecutionPlan) (execution.ExecutionPlan, error) {
	if err := execution.ValidatePlan(plan); err != nil {
		return execution.ExecutionPlan{}, err
	}
	if plan.CreatedAt.IsZero() {
		return execution.ExecutionPlan{}, fmt.Errorf("plan created time is required")
	}
	body, err := json.Marshal(plan)
	if err != nil {
		return execution.ExecutionPlan{}, fmt.Errorf("encode execution plan: %w", err)
	}

	err = r.withProjectTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		var taskID string
		err := tx.QueryRowContext(ctx, `SELECT task_id FROM execution_goals WHERE goal_id=$1`, string(plan.GoalRef)).Scan(&taskID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("resolve plan goal: %w", err)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO execution_plans (
			tenant_id, project_id, plan_id, task_id, goal_id, revision,
			parent_revision, plan_json, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			principal.TenantID, principal.ProjectID, string(plan.ID), taskID,
			string(plan.GoalRef), plan.Revision, plan.ParentRevision, body, plan.CreatedAt.UTC())
		if err != nil {
			return fmt.Errorf("create execution plan: %w", err)
		}
		return nil
	})
	if err != nil {
		return execution.ExecutionPlan{}, err
	}
	return plan, nil
}

func (r *ScopedPostgresExecutionLineage) GetPlan(ctx context.Context, planID execution.PlanID, revision int64) (execution.ExecutionPlan, error) {
	var plan execution.ExecutionPlan
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		var body []byte
		err := tx.QueryRowContext(ctx, `SELECT plan_json FROM execution_plans
			WHERE plan_id=$1 AND revision=$2`, string(planID), revision).Scan(&body)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get execution plan: %w", err)
		}
		if err := json.Unmarshal(body, &plan); err != nil {
			return fmt.Errorf("decode execution plan: %w", err)
		}
		return nil
	})
	return plan, err
}

func (r *ScopedPostgresExecutionLineage) ListPlansByTask(ctx context.Context, taskID string) ([]execution.ExecutionPlan, error) {
	items := make([]execution.ExecutionPlan, 0)
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		rows, err := tx.QueryContext(ctx, `SELECT plan_json FROM execution_plans
			WHERE task_id=$1 ORDER BY created_at, plan_id, revision`, taskID)
		if err != nil {
			return fmt.Errorf("list execution plans: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var body []byte
			if err := rows.Scan(&body); err != nil {
				return fmt.Errorf("scan execution plan: %w", err)
			}
			var plan execution.ExecutionPlan
			if err := json.Unmarshal(body, &plan); err != nil {
				return fmt.Errorf("decode execution plan: %w", err)
			}
			items = append(items, plan)
		}
		return rows.Err()
	})
	return items, err
}

func (r *ScopedPostgresExecutionLineage) CreateExecution(ctx context.Context, item execution.Execution) (execution.Execution, error) {
	if err := validateExecutionLineage(item); err != nil {
		return execution.Execution{}, err
	}
	policy, err := json.Marshal(item.PolicyContext)
	if err != nil {
		return execution.Execution{}, fmt.Errorf("encode execution policy context: %w", err)
	}
	budget, err := json.Marshal(item.Budget)
	if err != nil {
		return execution.Execution{}, fmt.Errorf("encode execution budget: %w", err)
	}
	status, err := json.Marshal(item.Status)
	if err != nil {
		return execution.Execution{}, fmt.Errorf("encode execution status: %w", err)
	}
	accounting, err := json.Marshal(item.Accounting)
	if err != nil {
		return execution.Execution{}, fmt.Errorf("encode execution accounting: %w", err)
	}

	err = r.withProjectTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		if item.Identity.Tenant == "" {
			item.Identity.Tenant = principal.TenantID
		}
		if item.Identity.Principal == "" {
			item.Identity.Principal = principal.SubjectID
		}
		if item.Identity.Tenant != principal.TenantID {
			return fmt.Errorf("execution tenant must match authenticated tenant")
		}

		var goalTaskID string
		if err := tx.QueryRowContext(ctx, `SELECT task_id FROM execution_goals WHERE goal_id=$1`, string(item.GoalRef)).Scan(&goalTaskID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("resolve execution goal: %w", err)
		}
		if goalTaskID != item.TaskRef {
			return fmt.Errorf("execution task does not match goal task")
		}
		var planGoalID, planTaskID string
		if err := tx.QueryRowContext(ctx, `SELECT goal_id, task_id FROM execution_plans
			WHERE plan_id=$1 AND revision=$2`, string(item.PlanRef.PlanID), item.PlanRef.Revision).Scan(&planGoalID, &planTaskID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("resolve execution plan: %w", err)
		}
		if planGoalID != string(item.GoalRef) || planTaskID != item.TaskRef {
			return fmt.Errorf("execution plan lineage does not match task/goal")
		}

		_, err := tx.ExecContext(ctx, `INSERT INTO executions (
			tenant_id, project_id, execution_id, task_id, goal_id, plan_id,
			plan_revision, principal, phase, policy_context, budget_state,
			status_json, accounting, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			principal.TenantID, principal.ProjectID, string(item.ID), item.TaskRef,
			string(item.GoalRef), string(item.PlanRef.PlanID), item.PlanRef.Revision,
			item.Identity.Principal, string(item.Status.Phase), policy, budget, status,
			accounting, item.CreatedAt.UTC(), item.UpdatedAt.UTC())
		if err != nil {
			return fmt.Errorf("create execution: %w", err)
		}
		return nil
	})
	if err != nil {
		return execution.Execution{}, err
	}
	return item, nil
}

func (r *ScopedPostgresExecutionLineage) GetExecution(ctx context.Context, executionID execution.ExecutionID) (execution.Execution, error) {
	var item execution.Execution
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		var tenantID, principal, phase string
		var policy, budget, status, accounting []byte
		err := tx.QueryRowContext(ctx, `SELECT tenant_id, execution_id, task_id, goal_id,
			plan_id, plan_revision, principal, phase, policy_context, budget_state,
			status_json, accounting, created_at, updated_at
			FROM executions WHERE execution_id=$1`, string(executionID)).Scan(
			&tenantID, &item.ID, &item.TaskRef, &item.GoalRef, &item.PlanRef.PlanID,
			&item.PlanRef.Revision, &principal, &phase, &policy, &budget, &status,
			&accounting, &item.CreatedAt, &item.UpdatedAt,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get execution: %w", err)
		}
		item.Identity = execution.Identity{Principal: principal, Tenant: tenantID}
		if err := json.Unmarshal(policy, &item.PolicyContext); err != nil {
			return fmt.Errorf("decode execution policy context: %w", err)
		}
		if err := json.Unmarshal(budget, &item.Budget); err != nil {
			return fmt.Errorf("decode execution budget: %w", err)
		}
		if err := json.Unmarshal(status, &item.Status); err != nil {
			return fmt.Errorf("decode execution status: %w", err)
		}
		if item.Status.Phase == "" {
			item.Status.Phase = execution.ExecutionPhase(phase)
		}
		if err := json.Unmarshal(accounting, &item.Accounting); err != nil {
			return fmt.Errorf("decode execution accounting: %w", err)
		}
		return nil
	})
	return item, err
}

func (r *ScopedPostgresExecutionLineage) ListExecutionsByTask(ctx context.Context, taskID string) ([]execution.Execution, error) {
	items := make([]execution.Execution, 0)
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		rows, err := tx.QueryContext(ctx, `SELECT tenant_id, execution_id, task_id, goal_id,
			plan_id, plan_revision, principal, phase, policy_context, budget_state,
			status_json, accounting, created_at, updated_at
			FROM executions WHERE task_id=$1 ORDER BY created_at, execution_id`, taskID)
		if err != nil {
			return fmt.Errorf("list executions: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanExecutionLineage(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func scanExecutionLineage(row scanner) (execution.Execution, error) {
	var item execution.Execution
	var tenantID, principal, phase string
	var policy, budget, status, accounting []byte
	if err := row.Scan(
		&tenantID, &item.ID, &item.TaskRef, &item.GoalRef, &item.PlanRef.PlanID,
		&item.PlanRef.Revision, &principal, &phase, &policy, &budget, &status,
		&accounting, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return execution.Execution{}, fmt.Errorf("scan execution: %w", err)
	}
	item.Identity = execution.Identity{Principal: principal, Tenant: tenantID}
	if err := json.Unmarshal(policy, &item.PolicyContext); err != nil {
		return execution.Execution{}, fmt.Errorf("decode execution policy context: %w", err)
	}
	if err := json.Unmarshal(budget, &item.Budget); err != nil {
		return execution.Execution{}, fmt.Errorf("decode execution budget: %w", err)
	}
	if err := json.Unmarshal(status, &item.Status); err != nil {
		return execution.Execution{}, fmt.Errorf("decode execution status: %w", err)
	}
	if item.Status.Phase == "" {
		item.Status.Phase = execution.ExecutionPhase(phase)
	}
	if err := json.Unmarshal(accounting, &item.Accounting); err != nil {
		return execution.Execution{}, fmt.Errorf("decode execution accounting: %w", err)
	}
	return item, nil
}

func validateGoalLineage(goal execution.Goal) error {
	if goal.ID == "" || strings.TrimSpace(goal.TaskRef) == "" || strings.TrimSpace(goal.Objective) == "" {
		return fmt.Errorf("goal id, task ref and objective are required")
	}
	return nil
}

func validateExecutionLineage(item execution.Execution) error {
	if item.ID == "" || strings.TrimSpace(item.TaskRef) == "" || item.GoalRef == "" ||
		item.PlanRef.PlanID == "" || item.PlanRef.Revision < 1 {
		return fmt.Errorf("execution id, task ref, goal ref and plan revision are required")
	}
	if item.Status.Phase == "" {
		return fmt.Errorf("execution phase is required")
	}
	if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() || item.UpdatedAt.Before(item.CreatedAt) {
		return fmt.Errorf("valid execution timestamps are required")
	}
	return nil
}

func requireScopedTask(ctx context.Context, tx *sql.Tx, taskID string) error {
	var found string
	err := tx.QueryRowContext(ctx, `SELECT id FROM tasks WHERE id=$1`, taskID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("resolve scoped task: %w", err)
	}
	return nil
}

func (r *ScopedPostgresExecutionLineage) withProjectTx(ctx context.Context, fn func(*sql.Tx, identity.Principal) error) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("execution lineage database is required")
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
		return fmt.Errorf("begin scoped execution transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return fmt.Errorf("set execution transaction scope: %w", err)
	}
	if err := fn(tx, principal); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scoped execution transaction: %w", err)
	}
	return nil
}

// Ensure time remains part of this file's persisted-domain contract even when
// compiler optimizations remove direct package-level references in tests.
var _ = time.Time{}
