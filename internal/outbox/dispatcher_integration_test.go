//go:build integration

package outbox

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
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestDispatcherRecoversCrashAfterPhysicalDeliveryWithoutDuplicateBusinessEffect(t *testing.T) {
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
	cleanupDispatcherFixture(t, ctx, db)
	defer cleanupDispatcherFixture(t, context.Background(), db)
	createDispatcherFixture(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "dispatcher",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	baseStore := repository.NewScopedPostgresOutbox(db)
	crashStore := &failFirstMarkDeliveredStore{Store: baseStore, fail: true}
	adapter := &sqlDeduplicatingAdapter{db: db}
	first, err := NewDispatcher(crashStore, "worker-before-crash", time.Minute, 5, time.Second)
	if err != nil {
		t.Fatalf("create first dispatcher: %v", err)
	}
	if err := first.Register("workflow.start", adapter); err != nil {
		t.Fatalf("register first adapter: %v", err)
	}
	start := time.Now().UTC()
	first.now = func() time.Time { return start }

	firstResult, firstErr := first.DispatchOnce(projectCtx, 10)
	if firstErr == nil || firstResult.Failed != 1 || firstResult.Delivered != 0 {
		t.Fatalf("simulated crash result=%#v err=%v", firstResult, firstErr)
	}
	var status domain.OutboxStatus
	if err := db.QueryRowContext(ctx, `SELECT status FROM outbox_messages WHERE outbox_id='outbox-crash'`).Scan(&status); err != nil {
		t.Fatalf("read outbox after simulated crash: %v", err)
	}
	if status != domain.OutboxDelivering {
		t.Fatalf("outbox status after simulated crash=%s want delivering", status)
	}
	assertDownstreamEffects(t, ctx, db, 1)
	if adapter.calls != 1 {
		t.Fatalf("physical delivery calls=%d want=1", adapter.calls)
	}

	second, err := NewDispatcher(baseStore, "worker-after-crash", time.Minute, 5, time.Second)
	if err != nil {
		t.Fatalf("create recovery dispatcher: %v", err)
	}
	if err := second.Register("workflow.start", adapter); err != nil {
		t.Fatalf("register recovery adapter: %v", err)
	}
	second.now = func() time.Time { return start.Add(2 * time.Minute) }
	secondResult, err := second.DispatchOnce(projectCtx, 10)
	if err != nil {
		t.Fatalf("recovery dispatch: %v", err)
	}
	if secondResult.Delivered != 1 || secondResult.Failed != 0 {
		t.Fatalf("recovery result=%#v", secondResult)
	}
	if adapter.calls != 2 {
		t.Fatalf("physical delivery calls=%d want=2 to prove at-least-once redelivery", adapter.calls)
	}
	assertDownstreamEffects(t, ctx, db, 1)
	if err := db.QueryRowContext(ctx, `SELECT status FROM outbox_messages WHERE outbox_id='outbox-crash'`).Scan(&status); err != nil {
		t.Fatalf("read recovered outbox: %v", err)
	}
	if status != domain.OutboxDelivered {
		t.Fatalf("recovered outbox status=%s want delivered", status)
	}
}

type failFirstMarkDeliveredStore struct {
	Store
	fail bool
}

func (s *failFirstMarkDeliveredStore) MarkDelivered(ctx context.Context, outboxID, owner string, at time.Time) error {
	if s.fail {
		s.fail = false
		return errors.New("simulated process crash after downstream delivery")
	}
	return s.Store.MarkDelivered(ctx, outboxID, owner, at)
}

type sqlDeduplicatingAdapter struct {
	db    *sql.DB
	calls int
}

func (a *sqlDeduplicatingAdapter) Deliver(ctx context.Context, message domain.OutboxMessage) error {
	a.calls++
	_, err := a.db.ExecContext(ctx, `INSERT INTO downstream_effects(idempotency_key, payload)
		VALUES ($1, $2::jsonb) ON CONFLICT (idempotency_key) DO NOTHING`,
		message.IdempotencyKey, string(message.Payload))
	return err
}

func createDispatcherFixture(t *testing.T, ctx context.Context, db *sql.DB) {
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
			CONSTRAINT outbox_dispatcher_status_check CHECK (status IN ('pending','delivering','delivered','dead_letter')),
			CONSTRAINT outbox_dispatcher_lease_check CHECK (
				(status='delivering' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
				OR (status<>'delivering' AND lease_owner IS NULL AND lease_expires_at IS NULL)
			)
		);
		CREATE TABLE downstream_effects (
			idempotency_key TEXT PRIMARY KEY,
			payload JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO outbox_messages(
			outbox_id, tenant_id, project_id, aggregate_type, aggregate_id, event_type,
			payload, destination, idempotency_key, status, attempts, available_at, created_at
		) VALUES (
			'outbox-crash','tenant-a','project-a','Task','task-1','TaskCreated',
			'{"taskId":"task-1"}'::jsonb,'workflow.start','delivery-crash-safe','pending',0,
			NOW()-INTERVAL '1 minute',NOW()-INTERVAL '1 minute'
		);
	`)
	if err != nil {
		t.Fatalf("create dispatcher integration fixture: %v", err)
	}
}

func cleanupDispatcherFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS downstream_effects CASCADE; DROP TABLE IF EXISTS outbox_messages CASCADE`)
}

func assertDownstreamEffects(t *testing.T, ctx context.Context, db *sql.DB, want int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM downstream_effects`).Scan(&count); err != nil {
		t.Fatalf("count downstream effects: %v", err)
	}
	if count != want {
		t.Fatalf("downstream business effects=%d want=%d", count, want)
	}
}
