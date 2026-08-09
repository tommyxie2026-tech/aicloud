package httpapi

import (
	"net/http"
	"strings"
)

// taskScopeGuard resolves a Task through the tenant-scoped repository before
// dispatching any Task resource or subresource. This makes route, cost, audit,
// model execution, trace, evaluation and tool endpoints inherit the same
// ownership boundary even when their underlying evidence stores are keyed only
// by task ID during the v0.1 migration.
func (s *Server) taskScopeGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/api/v1/tasks/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			next.ServeHTTP(w, r)
			return
		}
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
		if path == "" {
			next.ServeHTTP(w, r)
			return
		}
		taskID := strings.Split(path, "/")[0]
		if strings.Contains(taskID, ":") {
			taskID = strings.Split(taskID, ":")[0]
		}
		if taskID == "" {
			next.ServeHTTP(w, r)
			return
		}
		if _, err := s.control.GetTask(r.Context(), taskID); err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}
