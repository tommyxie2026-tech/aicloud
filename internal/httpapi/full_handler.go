package httpapi

import "net/http"

// FullHandler adds method-aware evidence and execution routes while retaining
// the existing resource handler as the compatibility fallback.
func (s *Server) FullHandler() http.Handler {
	mux := http.NewServeMux()
	s.registerEvidenceRoutes(mux)
	mux.Handle("/", s.Handler())
	return mux
}
