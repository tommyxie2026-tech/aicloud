package httpapi

import (
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func WithAuthorization(authorizer authorization.Authorizer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		principal, err := identity.RequirePrincipal(r.Context())
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authenticated principal is required", false, nil)
			return
		}
		request, methods, known := resolveAPIAccess(r, principal)
		if !known {
			writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "API resource not found", false, nil)
			return
		}
		if request.Action == "" {
			w.Header().Set("Allow", strings.Join(methods, ", "))
			writeAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method is not allowed", false, nil)
			return
		}
		if authorizer == nil {
			writeAPIError(w, http.StatusInternalServerError, "AUTHORIZATION_NOT_CONFIGURED", "authorization service is not configured", false, nil)
			return
		}
		decision, err := authorizer.Authorize(r.Context(), request)
		if err != nil {
			writeAPIError(w, http.StatusServiceUnavailable, "AUTHORIZATION_UNAVAILABLE", "authorization evaluation failed", true, nil)
			return
		}
		if !decision.Allowed {
			writeAPIError(w, http.StatusForbidden, "FORBIDDEN", "request is not authorized", false, nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
