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

func TestScopedPostgresOutboxDeadLetterRedriveIsBoundedAndAudited(t *testing.T) {
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
	cleanupOutboxRedriveFixture(ctx, db)
	defer cleanupOutboxRedriveFixture(context.Background(), db)
	createOutboxRedriveFixture(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "operator-a",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresOutbox(db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	leased, err := repo.Lease(projectCtx, "worker-before-redrive", 10, now, time.Minute)
	if err != nil {
		t.Fatalf("lease before redrive: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("dead-letter message was automatically revived: %#v", leased)
	}

	event, err := repo.RedriveDeadLetter(projectCtx, "outbox-dead-a", "operator reviewed downstream recovery", now, now)
	if err != nil {
		t.Fatalf("redrive dead letter: %v", err)
	}
	if event.TenantID != "tenant-a" || event.ProjectID != "project-a" || event.OutboxID != "outbox-dead-a" {
		t.Fatalf("unexpected redrive scope: %+v", event)
	}
	if event.ActorPrincipalType != string(identity.PrincipalUser) || event.ActorSubjectID != "operator-a" {
		t.Fatalf("unexpected redrive actor: %+v", event)
	}
	if event.PreviousAttempts != 3 || event.PreviousLastError != "retry budget exhausted" {
		t.Fatalf("unexpected previous failure evidence: %+v", event)
	}

	var status domain.OutboxStatus
	var attempts int
	var lastError string
	if err := db.QueryRowContext(ctx, `SELECT status, attempts, COALESCE(last_error,'') FROM outbox_messages WHERE outbox_id='outbox-dead-a'`).Scan(&status, &attempts, &lastError); err != nil {
		t.Fatalf("read redriven Outbox: %v", err)
	}
	if status != domain.OutboxPending || attempts != 3 || lastError != "retry budget exhausted" {
		t.Fatalf("redrive erased failure history: status=%s attempts=%d lastError=%q", status, attempts, lastError)
	}

	var evidenceCount int
	var storedReason string
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*), MIN(reason) FROM outbox_redrive_events WHERE outbox_id='outbox-dead-a'`).Scan(&evidenceCount, &storedReason); err != nil {
		t.Fatalf("read redrive evidence: %v", err)
	}
	if evidenceCount != 1 || storedReason != "operator reviewed downstream recovery" {
		t.Fatalf("redrive evidence count=%d reason=%q", evidenceCount, storedReason)
	}

	leased, err = repo.Lease(projectCtx, "worker-after-redrive", 10, now.Add(time.Second), time.Minute)
	if err != nil {
		t.Fatalf("lease after redrive: %v", err)
	}
	if len(leased) != 1 || leased[0].Message.OutboxID != "outbox-dead-a" || leased[0].Message.Attempts != 4 {
		t.Fatalf("redriven message did not re-enter normal delivery path: %#v", leased)
	}

	if _, err := repo.RedriveDeadLetter(projectCtx, "outbox-dead-b", "wrong project", now, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project redrive error=%v want ErrNotFound", err)
	}
	if _, err := repo.RedriveDeadLetter(projectCtx, "outbox-pending-a", "not a dead letter", now, now); !errors.Is(err, ErrOutboxNotDeadLetter) {
		t.Fatalf("pending redrive error=%v want ErrOutboxNotDeadLetter", err)
	}
}

func TestScopedPostgresOutboxRedriveRollsBackWhenEvidenceAppendFails(t *testing.T) {
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
	cleanupOutboxRedriveFixture(ctx, db)
	defer cleanupOutboxRedriveFixture(context.Background(), db)
	createOutboxRedriveFixture(t, ctx, db)

	if _, err := db.ExecContext(ctx, `DROP TABLE outbox_redrive_events`); err != nil {
		t.Fatalf("drop redrive evidence table: %v", err)
	}
	projectCtx := identity.WithPrincipal(ctx, identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "recovery-test",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "integration-test", Issuer: "test",
	})
	repo := NewScopedPostgresOutbox(db)
	now := time.Now().UTC()
	if _, err := repo.RedriveDeadLetter(projectCtx, "outbox-dead-a", "must roll back", now, now); err == nil {
		t.Fatal("redrive unexpectedly succeeded without evidence table")
	}

	var status domain.OutboxStatus
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT status, attempts FROM outbox_messages WHERE outbox_id='outbox-dead-a'`).Scan(&status, &attempts); err != nil {
		t.Fatalf("read rolled-back Outbox: %v", err)
	}
	if status != domain.OutboxDeadLetter || attempts != 3 {
		t.Fatalf("redrive status update escaped failed evidence transaction: status=%s attempts=%d", status, attempts)
	}
}

func createOutboxRedriveFixture(t *testing.T, ctx context.Context, db *sql.DB) {
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
    CONSTRAINT outbox_redrive_status_test_check CHECK (status IN ('pending','delivering','delivered','dead_letter')),
    CONSTRAINT outbox_redrive_lease_test_check CHECK (
        (status='delivering' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR (status<>'delivering' AND lease_owner IS NULL AND lease_expires_at IS NULL)
    )
);
CREATE TABLE outbox_redrive_events (
    redrive_event_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    outbox_id TEXT NOT NULL,
    actor_principal_type TEXT NOT NULL,
    actor_subject_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    previous_attempts INTEGER NOT NULL,
    previous_last_error TEXT NOT NULL DEFAULT '',
    redriven_at TIMESTAMPTZ NOT NULL
);
INSERT INTO outbox_messages(
    outbox_id, tenant_id, project_id, aggregate_type, aggregate_id, event_type,
    payload, destination, idempotency_key, status, attempts, available_at, created_at, last_error
) VALUES
    ('outbox-dead-a','tenant-a','project-a','Task','task-a','TaskCreated','{}'::jsonb,'workflow.start','dead-a','dead_letter',3,NOW(),NOW(),'retry budget exhausted'),
    ('outbox-dead-b','tenant-a','project-b','Task','task-b','TaskCreated','{}'::jsonb,'workflow.start','dead-b','dead_letter',3,NOW(),NOW(),'other project failure'),
    ('outbox-pending-a','tenant-a','project-a','Task','task-p','TaskCreated','{}'::jsonb,'workflow.start','pending-a','pending',0,NOW()+INTERVAL '1 hour',NOW(),NULL);
`)
	if err != nil {
		t.Fatalf("create Outbox redrive fixture: %v", err)
	}
}

func cleanupOutboxRedriveFixture(ctx context.Context, db *sql.DB) {
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS outbox_redrive_events CASCADE`)
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS outbox_messages CASCADE`)
}
