package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
)

type routeCommandRequest struct {
	RouteClass            domain.RouteClass      `json:"routeClass"`
	RequiredCapabilities  []string               `json:"requiredCapabilities,omitempty"`
	InferenceEffort       domain.InferenceEffort `json:"inferenceEffort,omitempty"`
	ServiceTier           domain.ServiceTier     `json:"serviceTier,omitempty"`
	EstimatedInputTokens  int                    `json:"estimatedInputTokens,omitempty"`
	EstimatedOutputTokens int                    `json:"estimatedOutputTokens,omitempty"`
	Budget                float64                `json:"budget,omitempty"`
	Currency              string                 `json:"currency,omitempty"`
	DataResidency         string                 `json:"dataResidency,omitempty"`
	EvidenceVersion       string                 `json:"evidenceVersion,omitempty"`
	PolicyVersion         string                 `json:"policyVersion,omitempty"`
	AllowDegraded         bool                   `json:"allowDegraded,omitempty"`
	RequireFreshSignals   bool                   `json:"requireFreshSignals,omitempty"`
	SignalMaxAgeSeconds   int                    `json:"signalMaxAgeSeconds,omitempty"`
}

func (s *Server) commandAwareTaskMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if taskID, ok := routeCommandTaskID(r.URL.Path); ok {
				s.routeTaskCommandAware(w, r, taskID)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func routeCommandTaskID(path string) (string, bool) {
	const prefix = "/api/v1/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "route" {
		return "", false
	}
	return parts[0], true
}

func (s *Server) routeTaskCommandAware(w http.ResponseWriter, r *http.Request, taskID string) {
	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" {
		writeErrorStatus(w, http.StatusBadRequest, "Idempotency-Key is required")
		return
	}
	var req routeCommandRequest
	if err := decodeJSON(r, &req); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	digest, err := canonicalRequestDigest(struct {
		TaskID  string              `json:"taskId"`
		Request routeCommandRequest `json:"request"`
	}{TaskID: taskID, Request: req})
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.control.DecideRouteIdempotent(r.Context(), router.Request{
		TaskID:                taskID,
		RouteClass:            req.RouteClass,
		RequiredCapabilities:  req.RequiredCapabilities,
		InferenceEffort:       req.InferenceEffort,
		ServiceTier:           req.ServiceTier,
		EstimatedInputTokens:  req.EstimatedInputTokens,
		EstimatedOutputTokens: req.EstimatedOutputTokens,
		Budget:                req.Budget,
		Currency:              req.Currency,
		DataResidency:         req.DataResidency,
		EvidenceVersion:       req.EvidenceVersion,
		PolicyVersion:         req.PolicyVersion,
		AllowDegraded:         req.AllowDegraded,
		RequireFreshSignals:   req.RequireFreshSignals,
		SignalMaxAge:          time.Duration(req.SignalMaxAgeSeconds) * time.Second,
	}, controlplane.CommandMetadata{
		IdempotencyKey: key,
		RequestDigest:  digest,
		RequestID:      strings.TrimSpace(r.Header.Get("X-Request-ID")),
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrIdempotencyConflict):
			writeErrorStatus(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
		case errors.Is(err, repository.ErrIdempotencyInProgress):
			writeErrorStatus(w, http.StatusConflict, "IDEMPOTENCY_IN_PROGRESS")
		case strings.Contains(err.Error(), "no policy-compliant model capacity"):
			writeErrorStatus(w, http.StatusServiceUnavailable, err.Error())
		case errors.Is(err, domain.ErrInvalidTaskTransition):
			writeErrorStatus(w, http.StatusConflict, err.Error())
		default:
			writeError(w, err)
		}
		return
	}
	if result.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(w, http.StatusCreated, result.Decision)
}
