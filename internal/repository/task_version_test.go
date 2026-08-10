package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestMemoryTasksOptimisticConcurrency(t *testing.T) {
	repo := NewMemoryTasks()
	now := time.Now().UTC()
	created, err := repo.Create(context.Background(), domain.Task{
		ID: "task-1", Input: "test", Status: domain.TaskCreated,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	stale := created
	created.Result = "first write"
	updated, err := repo.Update(context.Background(), created)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("version=%d want=2", updated.Version)
	}
	stale.Result = "stale write"
	if _, err := repo.Update(context.Background(), stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale Update error=%v want ErrVersionConflict", err)
	}
	stored, err := repo.Get(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Result != "first write" || stored.Version != 2 {
		t.Fatalf("stale write changed stored task: %#v", stored)
	}
}

func TestMemoryTasksDefaultsLegacyCreateToVersionOne(t *testing.T) {
	repo := NewMemoryTasks()
	created, err := repo.Create(context.Background(), domain.Task{ID: "task-legacy"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Version != 1 || created.Status != domain.TaskCreated {
		t.Fatalf("legacy create not normalized: %#v", created)
	}
}
