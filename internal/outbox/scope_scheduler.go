package outbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

const (
	OutboxDispatcherSubject     = "aicloud-outbox-dispatcher"
	OutboxDispatcherAuthnMethod = "internal_workload_identity"
	OutboxDispatcherIssuer      = "aicloud"
)

type ScopeProcessor interface {
	ProcessScope(context.Context, repository.OutboxDispatchScope) (int, error)
}

type ScopeProcessorFunc func(context.Context, repository.OutboxDispatchScope) (int, error)

func (f ScopeProcessorFunc) ProcessScope(ctx context.Context, scope repository.OutboxDispatchScope) (int, error) {
	return f(ctx, scope)
}

type ScopeScheduler struct {
	scopes    repository.OutboxDispatchScopeStore
	processor ScopeProcessor
	subject   string
}

type ScopeDispatchResult struct {
	ScopesAttempted int
	ScopesSucceeded int
	MessagesHandled int
}

func NewScopeScheduler(
	scopes repository.OutboxDispatchScopeStore,
	processor ScopeProcessor,
	subject string,
) (*ScopeScheduler, error) {
	if scopes == nil {
		return nil, fmt.Errorf("Outbox dispatch scope store is required")
	}
	if processor == nil {
		return nil, fmt.Errorf("Outbox scope processor is required")
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = OutboxDispatcherSubject
	}
	return &ScopeScheduler{scopes: scopes, processor: processor, subject: subject}, nil
}

func (s *ScopeScheduler) ProcessOnce(ctx context.Context, limit int) (ScopeDispatchResult, error) {
	if s == nil || s.scopes == nil || s.processor == nil {
		return ScopeDispatchResult{}, fmt.Errorf("Outbox scope scheduler is not configured")
	}
	scopes, err := s.scopes.List(ctx, limit)
	if err != nil {
		return ScopeDispatchResult{}, err
	}

	result := ScopeDispatchResult{}
	var failures []error
	for _, scope := range scopes {
		result.ScopesAttempted++
		principal := identity.Principal{
			Type:        identity.PrincipalServiceAccount,
			SubjectID:   s.subject,
			TenantID:    strings.TrimSpace(scope.TenantID),
			ProjectID:   strings.TrimSpace(scope.ProjectID),
			AuthnMethod: OutboxDispatcherAuthnMethod,
			Issuer:      OutboxDispatcherIssuer,
		}
		if err := identity.Validate(principal); err != nil || principal.ProjectID == "" {
			failures = append(failures, fmt.Errorf("invalid Outbox dispatch scope: tenant/project identity is incomplete"))
			continue
		}

		handled, err := s.processor.ProcessScope(identity.WithPrincipal(ctx, principal), scope)
		if err != nil {
			failures = append(failures, fmt.Errorf("dispatch Outbox scope %s/%s: %w", scope.TenantID, scope.ProjectID, err))
			continue
		}
		result.ScopesSucceeded++
		result.MessagesHandled += handled
	}
	return result, errors.Join(failures...)
}
