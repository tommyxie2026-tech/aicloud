package tenantrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/tenant"
)

type Ownership struct {
	TaskID    string
	TenantID  string
	ProjectID string
	SubjectID string
	CreatedAt time.Time
}

type OwnershipStore interface {
	Bind(context.Context, Ownership) error
	Get(context.Context, string) (Ownership, error)
}

type ScopedTasks struct {
	base       domain.TaskRepository
	ownership  OwnershipStore
}

func NewScopedTasks(base domain.TaskRepository, ownership OwnershipStore) *ScopedTasks {
	return &ScopedTasks{base: base, ownership: ownership}
}

func (r *ScopedTasks) List(ctx context.Context) ([]domain.Task, error) {
	items, err := r.base.List(ctx)
	if err != nil {
		return nil, err
	}
	scope, scoped := tenant.FromContext(ctx)
	if !scoped {
		return items, nil
	}
	filtered := make([]domain.Task, 0, len(items))
	for _, task := range items {
		owner, err := r.ownership.Get(ctx, task.ID)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if tenant.OwnsTask(scope, owner.TenantID, owner.ProjectID) {
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

func (r *ScopedTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	task, err := r.base.Get(ctx, id)
	if err != nil {
		return domain.Task{}, err
	}
	if err := r.authorize(ctx, id); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (r *ScopedTasks) Create(ctx context.Context, task domain.Task) (domain.Task, error) {
	created, err := r.base.Create(ctx, task)
	if err != nil {
		return domain.Task{}, err
	}
	scope, scoped := tenant.FromContext(ctx)
	if !scoped {
		return created, nil
	}
	if err := r.ownership.Bind(ctx, Ownership{
		TaskID: created.ID, TenantID: scope.TenantID, ProjectID: scope.ProjectID,
		SubjectID: scope.SubjectID, CreatedAt: time.Now().UTC(),
	}); err != nil {
		return domain.Task{}, fmt.Errorf("bind task tenant ownership: %w", err)
	}
	return created, nil
}

func (r *ScopedTasks) Update(ctx context.Context, task domain.Task) (domain.Task, error) {
	if err := r.authorize(ctx, task.ID); err != nil {
		return domain.Task{}, err
	}
	return r.base.Update(ctx, task)
}

func (r *ScopedTasks) authorize(ctx context.Context, taskID string) error {
	scope, scoped := tenant.FromContext(ctx)
	if !scoped {
		return nil
	}
	owner, err := r.ownership.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if !tenant.OwnsTask(scope, owner.TenantID, owner.ProjectID) {
		// Deliberately return not-found so callers cannot use authorization
		// failures to enumerate task identifiers across tenants.
		return repository.ErrNotFound
	}
	return nil
}

type MemoryOwnershipStore struct {
	mu sync.RWMutex
	m  map[string]Ownership
}

func NewMemoryOwnershipStore() *MemoryOwnershipStore {
	return &MemoryOwnershipStore{m: make(map[string]Ownership)}
}

func (s *MemoryOwnershipStore) Bind(_ context.Context, ownership Ownership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.m[ownership.TaskID]; ok {
		if current.TenantID == ownership.TenantID && current.ProjectID == ownership.ProjectID {
			return nil
		}
		return fmt.Errorf("task ownership already exists")
	}
	s.m[ownership.TaskID] = ownership
	return nil
}

func (s *MemoryOwnershipStore) Get(_ context.Context, taskID string) (Ownership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ownership, ok := s.m[taskID]
	if !ok {
		return Ownership{}, repository.ErrNotFound
	}
	return ownership, nil
}

type PostgresOwnershipStore struct{ db *sql.DB }

func NewPostgresOwnershipStore(db *sql.DB) *PostgresOwnershipStore {
	return &PostgresOwnershipStore{db: db}
}

func (s *PostgresOwnershipStore) Bind(ctx context.Context, ownership Ownership) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_ownership (
		task_id, tenant_id, project_id, subject_id, created_at
	) VALUES ($1,$2,$3,$4,$5)
	ON CONFLICT (task_id) DO NOTHING`, ownership.TaskID, ownership.TenantID,
		ownership.ProjectID, ownership.SubjectID, ownership.CreatedAt)
	if err != nil {
		return fmt.Errorf("bind task ownership: %w", err)
	}
	current, err := s.Get(ctx, ownership.TaskID)
	if err != nil {
		return err
	}
	if current.TenantID != ownership.TenantID || current.ProjectID != ownership.ProjectID {
		return fmt.Errorf("task ownership conflict")
	}
	return nil
}

func (s *PostgresOwnershipStore) Get(ctx context.Context, taskID string) (Ownership, error) {
	var ownership Ownership
	err := s.db.QueryRowContext(ctx, `SELECT task_id, tenant_id, project_id, subject_id, created_at
		FROM task_ownership WHERE task_id=$1`, taskID).Scan(
		&ownership.TaskID, &ownership.TenantID, &ownership.ProjectID,
		&ownership.SubjectID, &ownership.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Ownership{}, repository.ErrNotFound
	}
	if err != nil {
		return Ownership{}, fmt.Errorf("get task ownership: %w", err)
	}
	return ownership, nil
}

// SortedTaskIDs is intentionally small and test-oriented; production list
// paths can later push this filtering into SQL without changing ScopedTasks.
func SortedTaskIDs(items []Ownership) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.TaskID)
	}
	sort.Strings(ids)
	return ids
}
