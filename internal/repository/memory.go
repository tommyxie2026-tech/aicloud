package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var (
	ErrNotFound        = errors.New("resource not found")
	ErrVersionConflict = errors.New("resource version conflict")
)

type MemoryModels struct {
	mu sync.RWMutex
	m  map[string]domain.Model
}

func NewMemoryModels(seed ...domain.Model) *MemoryModels {
	r := &MemoryModels{m: make(map[string]domain.Model, len(seed))}
	for _, model := range seed {
		r.m[model.ID] = model
	}
	return r
}

func (r *MemoryModels) List(_ context.Context) ([]domain.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Model, 0, len(r.m))
	for _, model := range r.m {
		items = append(items, model)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r *MemoryModels) Get(_ context.Context, id string) (domain.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.m[id]
	if !ok {
		return domain.Model{}, ErrNotFound
	}
	return model, nil
}

func (r *MemoryModels) Create(_ context.Context, model domain.Model) (domain.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[model.ID]; ok {
		return domain.Model{}, errors.New("model already exists")
	}
	r.m[model.ID] = model
	return model, nil
}

func (r *MemoryModels) Update(_ context.Context, model domain.Model) (domain.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[model.ID]; !ok {
		return domain.Model{}, ErrNotFound
	}
	r.m[model.ID] = model
	return model, nil
}

type MemoryTasks struct {
	mu sync.RWMutex
	m  map[string]domain.Task
}

func NewMemoryTasks() *MemoryTasks { return &MemoryTasks{m: make(map[string]domain.Task)} }

func (r *MemoryTasks) List(_ context.Context) ([]domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.Task, 0, len(r.m))
	for _, task := range r.m {
		items = append(items, task)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryTasks) Get(_ context.Context, id string) (domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.m[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return task, nil
}

func (r *MemoryTasks) Create(_ context.Context, task domain.Task) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[task.ID]; ok {
		return domain.Task{}, errors.New("task already exists")
	}
	if task.Version == 0 {
		task.Version = 1
	}
	if task.Status == "" {
		task.Status = domain.TaskCreated
	}
	r.m[task.ID] = task
	return task, nil
}

func (r *MemoryTasks) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.m[task.ID]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	if task.Version != current.Version {
		return domain.Task{}, ErrVersionConflict
	}
	task.Version++
	r.m[task.ID] = task
	return task, nil
}

type MemoryRouteDecisions struct {
	mu sync.RWMutex
	m  map[string]domain.RouteDecision
}

func NewMemoryRouteDecisions() *MemoryRouteDecisions {
	return &MemoryRouteDecisions{m: make(map[string]domain.RouteDecision)}
}

func (r *MemoryRouteDecisions) Create(_ context.Context, decision domain.RouteDecision) (domain.RouteDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[decision.ID]; ok {
		return domain.RouteDecision{}, errors.New("route decision already exists")
	}
	r.m[decision.ID] = decision
	return decision, nil
}

func (r *MemoryRouteDecisions) Get(_ context.Context, id string) (domain.RouteDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decision, ok := r.m[id]
	if !ok {
		return domain.RouteDecision{}, ErrNotFound
	}
	return decision, nil
}

func (r *MemoryRouteDecisions) ListByTask(_ context.Context, taskID string) ([]domain.RouteDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.RouteDecision, 0)
	for _, decision := range r.m {
		if decision.TaskID == taskID {
			items = append(items, decision)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

type MemoryCostEvents struct {
	mu     sync.RWMutex
	events []domain.CostEvent
}

func NewMemoryCostEvents() *MemoryCostEvents { return &MemoryCostEvents{} }

func (r *MemoryCostEvents) Append(_ context.Context, event domain.CostEvent) (domain.CostEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return event, nil
}

func (r *MemoryCostEvents) ListByTask(_ context.Context, taskID string) ([]domain.CostEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]domain.CostEvent, 0)
	for _, event := range r.events {
		if event.TaskID == taskID {
			items = append(items, event)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
