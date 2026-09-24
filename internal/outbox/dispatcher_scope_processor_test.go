package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

type scopeProcessorStore struct {
	leaseCalls int
	principal  identity.Principal
}

func (s *scopeProcessorStore) Lease(ctx context.Context, _ string, _ int, _ time.Time, _ time.Duration) ([]repository.LeasedOutboxMessage, error) {
	s.leaseCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return nil, err
	}
	s.principal = principal
	return nil, nil
}

func (*scopeProcessorStore) MarkDelivered(context.Context, string, string, time.Time) error {
	return nil
}

func (*scopeProcessorStore) FailDelivery(context.Context, string, string, time.Time, time.Time, int, string) (domain.OutboxStatus, error) {
	return domain.OutboxPending, nil
}

func TestDispatcherScopeProcessorUsesExistingProjectPrincipal(t *testing.T) {
	store := &scopeProcessorStore{}
	dispatcher, err := NewDispatcher(store, "worker-a", time.Minute, 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewDispatcherScopeProcessor(dispatcher, 25)
	if err != nil {
		t.Fatal(err)
	}
	principal := identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: OutboxDispatcherSubject,
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: OutboxDispatcherAuthnMethod, Issuer: OutboxDispatcherIssuer,
	}
	ctx := identity.WithPrincipal(context.Background(), principal)
	handled, err := processor.ProcessScope(ctx, repository.OutboxDispatchScope{TenantID: "tenant-a", ProjectID: "project-a"})
	if err != nil {
		t.Fatal(err)
	}
	if handled != 0 || store.leaseCalls != 1 {
		t.Fatalf("handled=%d leaseCalls=%d", handled, store.leaseCalls)
	}
	if store.principal.TenantID != "tenant-a" || store.principal.ProjectID != "project-a" || store.principal.SubjectID != OutboxDispatcherSubject {
		t.Fatalf("unexpected scoped principal: %+v", store.principal)
	}
}

func TestDispatcherScopeProcessorRejectsScopeMismatchBeforeLease(t *testing.T) {
	store := &scopeProcessorStore{}
	dispatcher, err := NewDispatcher(store, "worker-a", time.Minute, 3, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewDispatcherScopeProcessor(dispatcher, 25)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: OutboxDispatcherSubject,
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: OutboxDispatcherAuthnMethod, Issuer: OutboxDispatcherIssuer,
	})
	if _, err := processor.ProcessScope(ctx, repository.OutboxDispatchScope{TenantID: "tenant-a", ProjectID: "project-b"}); err == nil {
		t.Fatal("mismatched scope was accepted")
	}
	if store.leaseCalls != 0 {
		t.Fatalf("scope mismatch reached Outbox Lease, calls=%d", store.leaseCalls)
	}
}
