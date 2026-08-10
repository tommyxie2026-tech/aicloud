//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestScopedPostgresTasksRejectsStaleVersion(t *testing.T) {
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS tasks CASCADE`)
	defer db.ExecContext(context.Background(), `DROP TABLE IF EXISTS tasks CASCADE`)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			created_by TEXT NOT NULL,
			agent_id TEXT NOT NULL DEFAULT '',
			input TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
			result TEXT NOT NULL DEFAULT '',
			cost DOUBLE PRECISION NOT NULL DEFAULT 0,
			estimated_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
			actual_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD',
			route_decision_id TEXT NOT NULL DEFAULT '',
			trace_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			completed_at TIMESTAMPTZ
		);
	`)
	if err != nil {
		t.Fatalf("create tasks fixture: %v", err)
	}

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresTasks(db)
	now := time.Now().UTC()
	created, err := repo.Create(projectCtx, domain.Task{
		ID: "task-1", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		Input: "test", Status: domain.TaskCreated, Version: 1, Currency: "USD",
		TraceID: "trace-1", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	first := created
	stale := created
	first.Result = "first"
	first.UpdatedAt = now.Add(time.Second)
	updated, err := repo.Update(projectCtx, first)
	if err != nil {
		t.Fatalf("first Update returned error: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("updated version=%d want=2", updated.Version)
	}

	stale.Result = "stale"
	stale.UpdatedAt = now.Add(2 * time.Second)
	if _, err := repo.Update(projectCtx, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale Update error=%v want ErrVersionConflict", err)
	}

	stored, err := repo.Get(projectCtx, created.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if stored.Result != "first" || stored.Version != 2 {
		t.Fatalf("stale write changed task: %#v", stored)
	}
}
