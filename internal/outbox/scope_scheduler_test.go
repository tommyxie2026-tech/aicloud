package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

type memoryScopeStore struct {
	items []repository.OutboxDispatchScope
	err   error
}

func (s *memoryScopeStore) List(context.Context, int) ([]repository.OutboxDispatchScope, error) {
	return append([]repository.OutboxDispatchScope(nil), s.items...), s.err
}

func TestScopeSchedulerBindsServiceAccountPerProject(t *testing.T) {
	now := time.Now().UTC()
	store := &memoryScopeStore{items: []repository.OutboxDispatchScope{
		{TenantID: "tenant-a", ProjectID: "project-a", FirstSeenAt: now, LastSeenAt: now},
		{TenantID: "tenant-b", ProjectID: "project-b", FirstSeenAt: now, LastSeenAt: now},
	}}
	seen := make([]identity.Principal, 0, 2)
	processor := ScopeProcessorFunc(func(ctx context.Context, scope repository.OutboxDispatchScope) (int, error) {
		principal, err := identity.RequireProject(ctx)
		if err != nil {
			return 0, err
		}
		seen = append(seen, principal)
		if principal.TenantID != scope.TenantID || principal.ProjectID != scope.ProjectID {
			t.Fatalf("principal=%+v scope=%+v", principal, scope)
		}
		return 2, nil
	})
	scheduler, err := NewScopeScheduler(store, processor, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := scheduler.ProcessOnce(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if result.ScopesAttempted != 2 || result.ScopesSucceeded != 2 || result.MessagesHandled != 4 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(seen) != 2 {
		t.Fatalf("principals=%d", len(seen))
	}
	for _, principal := range seen {
		if principal.Type != identity.PrincipalServiceAccount || principal.SubjectID != OutboxDispatcherSubject || principal.AuthnMethod != OutboxDispatcherAuthnMethod {
			t.Fatalf("unexpected dispatcher principal: %+v", principal)
		}
	}
}

func TestScopeSchedulerContinuesAfterTenantFailure(t *testing.T) {
	store := &memoryScopeStore{items: []repository.OutboxDispatchScope{
		{TenantID: "tenant-a", ProjectID: "project-a"},
		{TenantID: "tenant-b", ProjectID: "project-b"},
	}}
	calls := 0
	sentinel := errors.New("tenant-a temporarily unavailable")
	processor := ScopeProcessorFunc(func(_ context.Context, scope repository.OutboxDispatchScope) (int, error) {
		calls++
		if scope.TenantID == "tenant-a" {
			return 0, sentinel
		}
		return 1, nil
	})
	scheduler, err := NewScopeScheduler(store, processor, "dispatcher-test")
	if err != nil {
		t.Fatal(err)
	}
	result, err := scheduler.ProcessOnce(context.Background(), 100)
	if !errors.Is(err, sentinel) {
		t.Fatalf("error=%v", err)
	}
	if calls != 2 {
		t.Fatalf("one tenant failure blocked later scopes, calls=%d", calls)
	}
	if result.ScopesAttempted != 2 || result.ScopesSucceeded != 1 || result.MessagesHandled != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestScopeSchedulerRejectsInvalidGlobalScopeWithoutProcessing(t *testing.T) {
	store := &memoryScopeStore{items: []repository.OutboxDispatchScope{{TenantID: "tenant-a", ProjectID: ""}}}
	calls := 0
	processor := ScopeProcessorFunc(func(context.Context, repository.OutboxDispatchScope) (int, error) {
		calls++
		return 0, nil
	})
	scheduler, err := NewScopeScheduler(store, processor, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := scheduler.ProcessOnce(context.Background(), 100)
	if err == nil {
		t.Fatal("invalid global scope was accepted")
	}
	if calls != 0 || result.ScopesAttempted != 1 || result.ScopesSucceeded != 0 {
		t.Fatalf("calls=%d result=%+v", calls, result)
	}
}

func TestNewScopeSchedulerRequiresReadOnlyScopeSourceAndProcessor(t *testing.T) {
	processor := ScopeProcessorFunc(func(context.Context, repository.OutboxDispatchScope) (int, error) { return 0, nil })
	if _, err := NewScopeScheduler(nil, processor, ""); err == nil {
		t.Fatal("nil scope store accepted")
	}
	if _, err := NewScopeScheduler(&memoryScopeStore{}, nil, ""); err == nil {
		t.Fatal("nil processor accepted")
	}
}
