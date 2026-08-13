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

func TestScopedPostgresOutboxLeaseRecoveryRetryAndDeadLetter(t *testing.T) {
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
	cleanupOutboxFixture(t, ctx, db)
	defer cleanupOutboxFixture(t, context.Background(), db)
	createOutboxFixture(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "dispatcher-a",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresOutbox(db)
	now := time.Now().UTC()

	leased, err := repo.Lease(projectCtx, "worker-1", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("first lease: %v", err)
	}
	if len(leased) != 1 || leased[0].Message.OutboxID != "outbox-retry" || leased[0].Message.Attempts != 1 {
		t.Fatalf("unexpected first lease: %#v", leased)
	}

	leased, err = repo.Lease(projectCtx, "worker-2", 10, now.Add(30*time.Second), time.Minute)
	if err != nil {
		t.Fatalf("lease before expiry: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("active lease must not be stolen: %#v", leased)
	}

	leased, err = repo.Lease(projectCtx, "worker-2", 10, now.Add(2*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("lease after expiry: %v", err)
	}
	if len(leased) != 1 || leased[0].Message.Attempts != 2 || leased[0].LeaseOwner != "worker-2" {
		t.Fatalf("expired lease was not recovered: %#v", leased)
	}
	if err := repo.MarkDelivered(projectCtx, "outbox-retry", "worker-1", now.Add(2*time.Minute)); !errors.Is(err, ErrOutboxLeaseLost) {
		t.Fatalf("old lease owner error=%v want ErrOutboxLeaseLost", err)
	}

	status, err := repo.FailDelivery(projectCtx, "outbox-retry", "worker-2", now.Add(2*time.Minute), now.Add(3*time.Minute), 3, "temporary downstream failure")
	if err != nil {
		t.Fatalf("retry failure: %v", err)
	}
	if status != domain.OutboxPending {
		t.Fatalf("failure status=%s want pending", status)
	}

	leased, err = repo.Lease(projectCtx, "worker-3", 10, now.Add(4*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("third lease: %v", err)
	}
	if len(leased) != 1 || leased[0].Message.Attempts != 3 {
		t.Fatalf("unexpected third lease: %#v", leased)
	}
	status, err = repo.FailDelivery(projectCtx, "outbox-retry", "worker-3", now.Add(4*time.Minute), now.Add(5*time.Minute), 3, "terminal retry budget exhausted")
	if err != nil {
		t.Fatalf("dead letter failure: %v", err)
	}
	if status != domain.OutboxDeadLetter {
		t.Fatalf("failure status=%s want dead_letter", status)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO outbox_messages(
		outbox_id, tenant_id, project_id, aggregate_type, aggregate_id, event_type,
		payload, destination, idempotency_key, status, attempts, available_at, created_at
	) VALUES ('outbox-success','tenant-a','project-a','Task','task-2','TaskCreated','{}'::jsonb,
		'workflow.start','delivery-success','pending',0,$1,$1)`, now); err != nil {
		t.Fatalf("insert successful outbox fixture: %v", err)
	}
	leased, err = repo.Lease(projectCtx, "worker-4", 10, now.Add(10*time.Minute), time.Minute)
	if err != nil {
		t.Fatalf("success lease: %v", err)
	}
	if len(leased) != 1 || leased[0].Message.OutboxID != "outbox-success" {
		t.Fatalf("unexpected success lease: %#v", leased)
	}
	if err := repo.MarkDelivered(projectCtx, "outbox-success", "worker-4", now.Add(10*time.Minute)); err != nil {
		t.Fatalf("mark delivered: %v", err)
	}
	var storedStatus domain.OutboxStatus
	var deliveredAt sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT status, delivered_at FROM outbox_messages WHERE outbox_id='outbox-success'`).Scan(&storedStatus, &deliveredAt); err != nil {
		t.Fatalf("read delivered message: %v", err)
	}
	if storedStatus != domain.OutboxDelivered || !deliveredAt.Valid {
		t.Fatalf("stored status=%s delivered_at=%v", storedStatus, deliveredAt)
	}
}

func createOutboxFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE outbox_messages (
			outbox_id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			task_id TEXT,
			aggregate_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			payload JSONB NOT NULL,
			destination TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL,
			available_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			delivered_at TIMESTAMPTZ,
			lease_owner TEXT,
			lease_expires_at TIMESTAMPTZ,
			last_error TEXT,
			CONSTRAINT outbox_status_test_check CHECK (status IN ('pending','delivering','delivered','dead_letter')),
			CONSTRAINT outbox_lease_test_check CHECK (
				(status='delivering' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
				OR (status<>'delivering' AND lease_owner IS NULL AND lease_expires_at IS NULL)
			)
		);
		INSERT INTO outbox_messages(
			outbox_id, tenant_id, project_id, aggregate_type, aggregate_id, event_type,
			payload, destination, idempotency_key, status, attempts, available_at, created_at
		) VALUES ('outbox-retry','tenant-a','project-a','Task','task-1','TaskCreated','{}'::jsonb,
			'workflow.start','delivery-retry','pending',0,NOW()-INTERVAL '1 minute',NOW()-INTERVAL '1 minute');
	`)
	if err != nil {
		t.Fatalf("create outbox fixture: %v", err)
	}
}

func cleanupOutboxFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS outbox_messages CASCADE`)
}
