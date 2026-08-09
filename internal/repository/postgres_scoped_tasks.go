package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

const scopedTaskColumns = `id, tenant_id, project_id, created_by, agent_id, input, status,
	result, cost, estimated_cost, actual_cost, currency, route_decision_id, trace_id,
	created_at, updated_at`

// ValidateRuntimeDatabaseRole prevents API/Worker processes from starting with
// PostgreSQL credentials that can bypass RLS. Administrative and migration
// credentials must use separate entry points.
func ValidateRuntimeDatabaseRole(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	var role string
	var superuser, bypassRLS bool
	if err := db.QueryRowContext(ctx, `SELECT current_user, rolsuper, rolbypassrls
		FROM pg_roles WHERE rolname = current_user`).Scan(&role, &superuser, &bypassRLS); err != nil {
		return fmt.Errorf("inspect PostgreSQL runtime role: %w", err)
	}
	if superuser || bypassRLS {
		return fmt.Errorf("runtime PostgreSQL role %q must not be superuser or BYPASSRLS", role)
	}
	return nil
}

// ScopedPostgresTasks is the production runtime Task repository. Every
// operation requires a verified project-scoped Principal and sets tenant/project
// context transaction-locally before issuing SQL. PostgreSQL RLS is the second
// line of defense behind application authorization.
type ScopedPostgresTasks struct {
	db *sql.DB
}

func NewScopedPostgresTasks(db *sql.DB) *ScopedPostgresTasks {
	return &ScopedPostgresTasks{db: db}
}

func (r *ScopedPostgresTasks) List(ctx context.Context) ([]domain.Task, error) {
	var items []domain.Task
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		rows, err := tx.QueryContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks ORDER BY created_at`)
		if err != nil {
			return fmt.Errorf("list scoped tasks: %w", err)
		}
		defer rows.Close()
		items = make([]domain.Task, 0)
		for rows.Next() {
			task, err := scanScopedTask(rows)
			if err != nil {
				return err
			}
			items = append(items, task)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate scoped tasks: %w", err)
		}
		return nil
	})
	return items, err
}

func (r *ScopedPostgresTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	var task domain.Task
	err := r.withProjectTx(ctx, func(tx *sql.Tx, _ identity.Principal) error {
		var err error
		task, err = scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1`, id))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get scoped task: %w", err)
		}
		return nil
	})
	return task, err
}

func (r *ScopedPostgresTasks) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	err := r.withProjectTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		if task.TenantID != principal.TenantID || task.ProjectID != principal.ProjectID || task.CreatedBy != principal.SubjectID {
			return fmt.Errorf("task identity must match authenticated principal")
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO tasks (
			id, tenant_id, project_id, created_by, agent_id, input, status, result,
			cost, estimated_cost, actual_cost, currency, route_decision_id, trace_id,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			task.ID, task.TenantID, task.ProjectID, task.CreatedBy, task.AgentID,
			task.Input, task.Status, task.Result, task.Cost, task.EstimatedCost,
			task.ActualCost, task.Currency, task.RouteDecisionID, task.TraceID,
			task.CreatedAt, task.UpdatedAt)
		if err != nil {
			return fmt.Errorf("create scoped task: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (r *ScopedPostgresTasks) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	err := r.withProjectTx(ctx, func(tx *sql.Tx, principal identity.Principal) error {
		current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, task.ID))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load scoped task for update: %w", err)
		}
		if current.TenantID != task.TenantID || current.ProjectID != task.ProjectID || current.CreatedBy != task.CreatedBy {
			return fmt.Errorf("task tenant, project and creator identity are immutable")
		}
		if !principal.OwnsProject(current.TenantID, current.ProjectID) {
			return ErrNotFound
		}
		result, err := tx.ExecContext(ctx, `UPDATE tasks SET agent_id=$2, input=$3,
			status=$4, result=$5, cost=$6, estimated_cost=$7, actual_cost=$8,
			currency=$9, route_decision_id=$10, trace_id=$11, updated_at=$12 WHERE id=$1`,
			task.ID, task.AgentID, task.Input, task.Status, task.Result, task.Cost,
			task.EstimatedCost, task.ActualCost, task.Currency, task.RouteDecisionID,
			task.TraceID, task.UpdatedAt)
		if err != nil {
			return fmt.Errorf("update scoped task: %w", err)
		}
		return requireAffected(result)
	})
	if err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (r *ScopedPostgresTasks) withProjectTx(ctx context.Context, fn func(*sql.Tx, identity.Principal) error) error {
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scoped task transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return fmt.Errorf("set task transaction scope: %w", err)
	}
	if err := fn(tx, principal); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scoped task transaction: %w", err)
	}
	return nil
}

func scanScopedTask(row scanner) (domain.Task, error) {
	var task domain.Task
	if err := row.Scan(
		&task.ID, &task.TenantID, &task.ProjectID, &task.CreatedBy,
		&task.AgentID, &task.Input, &task.Status, &task.Result, &task.Cost,
		&task.EstimatedCost, &task.ActualCost, &task.Currency,
		&task.RouteDecisionID, &task.TraceID, &task.CreatedAt, &task.UpdatedAt,
	); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

// AdminPostgresTasks is intentionally separate from the runtime repository.
// It assumes db uses an independently managed admin credential and still
// requires an explicit System Principal with database:admin capability.
type AdminPostgresTasks struct {
	db *sql.DB
}

func OpenAdminPostgresTasks(ctx context.Context, dsn string) (*AdminPostgresTasks, func() error, error) {
	if dsn == "" {
		return nil, nil, fmt.Errorf("admin database URL is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open admin PostgreSQL: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping admin PostgreSQL: %w", err)
	}
	var role string
	var superuser, bypassRLS bool
	if err := db.QueryRowContext(ctx, `SELECT current_user, rolsuper, rolbypassrls
		FROM pg_roles WHERE rolname = current_user`).Scan(&role, &superuser, &bypassRLS); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("inspect admin PostgreSQL role: %w", err)
	}
	if !superuser && !bypassRLS {
		_ = db.Close()
		return nil, nil, fmt.Errorf("admin PostgreSQL role %q must be an independently managed RLS-bypass role", role)
	}
	return &AdminPostgresTasks{db: db}, db.Close, nil
}

func (r *AdminPostgresTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	if _, err := identity.RequireSystemCapability(ctx, identity.CapabilityDatabaseAdmin); err != nil {
		return domain.Task{}, err
	}
	task, err := scanScopedTask(r.db.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("admin get task: %w", err)
	}
	return task, nil
}

func (r *AdminPostgresTasks) List(ctx context.Context) ([]domain.Task, error) {
	if _, err := identity.RequireSystemCapability(ctx, identity.CapabilityDatabaseAdmin); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("admin list tasks: %w", err)
	}
	defer rows.Close()
	items := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanScopedTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, task)
	}
	return items, rows.Err()
}

// Compile-time guard that AdminPostgresTasks is read-only by design in S1.
var _ = time.Time{}
