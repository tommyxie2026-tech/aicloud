package repository

// TaskCommands exposes the R6 PostgreSQL transaction kernel from the scoped
// Task repository used by the production runtime.
func (r *ScopedPostgresTasks) TaskCommands() TaskCommandStore {
	if r == nil || r.db == nil {
		return nil
	}
	return NewScopedPostgresTaskCommands(r.db)
}
