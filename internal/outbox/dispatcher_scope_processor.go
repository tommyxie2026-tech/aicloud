package outbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

const defaultScopeDispatchBatchSize = 100

type DispatcherScopeProcessor struct {
	dispatcher *Dispatcher
	batchSize  int
}

func NewDispatcherScopeProcessor(dispatcher *Dispatcher, batchSize int) (*DispatcherScopeProcessor, error) {
	if dispatcher == nil {
		return nil, fmt.Errorf("Outbox dispatcher is required")
	}
	if batchSize <= 0 {
		batchSize = defaultScopeDispatchBatchSize
	}
	if batchSize > 1000 {
		batchSize = 1000
	}
	return &DispatcherScopeProcessor{dispatcher: dispatcher, batchSize: batchSize}, nil
}

func (p *DispatcherScopeProcessor) ProcessScope(ctx context.Context, scope repository.OutboxDispatchScope) (int, error) {
	if p == nil || p.dispatcher == nil {
		return 0, fmt.Errorf("Outbox dispatcher scope processor is not configured")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return 0, err
	}
	tenantID := strings.TrimSpace(scope.TenantID)
	projectID := strings.TrimSpace(scope.ProjectID)
	if tenantID == "" || projectID == "" || principal.TenantID != tenantID || principal.ProjectID != projectID {
		return 0, fmt.Errorf("Outbox dispatch scope does not match current project principal")
	}
	result, err := p.dispatcher.DispatchOnce(ctx, p.batchSize)
	return result.Leased, err
}
