package tenantrepo

import "github.com/tommyxie2026-tech/aicloud/internal/repository"

// TaskCommands propagates the optional R6 transaction kernel through the
// tenant-scoped Task repository wrapper. The wrapper remains responsible for
// ordinary TaskRepository access, while command atomicity is delegated to the
// underlying durable store.
func (r *ScopedTasks) TaskCommands() repository.TaskCommandStore {
	if r == nil || r.base == nil {
		return nil
	}
	provider, ok := r.base.(repository.TaskCommandStoreProvider)
	if !ok {
		return nil
	}
	return provider.TaskCommands()
}
