package httpapi

import "net/http"

// FullHandler adds method-aware evidence and execution routes while retaining
// the existing resource handler as the compatibility fallback. Task resources
// are resolved through the scoped Task repository before any subresource store
// is accessed. Public Task creation is intercepted here so it always carries
// the R6 command-idempotency contract.
func (s *Server) FullHandler() http.Handler {
	mux := http.NewServeMux()
	s.registerEvidenceRoutes(mux)
	mux.HandleFunc("/api/v1/tasks", s.taskCollectionCommandAware)
	mux.Handle("/", s.Handler())
	return s.taskScopeGuard(mux)
}
