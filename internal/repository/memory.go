package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var ErrNotFound = errors.New("resource not found")

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
	r.m[task.ID] = task
	return task, nil
}
func (r *MemoryTasks) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[task.ID]; !ok {
		return domain.Task{}, ErrNotFound
	}
	r.m[task.ID] = task
	return task, nil
}
