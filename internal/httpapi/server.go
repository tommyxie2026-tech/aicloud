package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
)

type Server struct {
	control *controlplane.Service
	log     *slog.Logger
}

func New(control *controlplane.Service, log *slog.Logger) *Server {
	return &Server{control: control, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.HandleFunc("/api/v1/models", s.models)
	mux.HandleFunc("/api/v1/models/", s.modelByID)
	mux.HandleFunc("/api/v1/tasks", s.tasks)
	mux.HandleFunc("/api/v1/tasks/", s.taskResource)
	return requestLogger(s.log, mux)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.control.ListModels(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var model domain.Model
		if err := decodeJSON(r, &model); err != nil {
			writeErrorStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		if model.ID == "" || model.Name == "" || model.Provider == "" {
			writeErrorStatus(w, http.StatusBadRequest, "id, name and provider are required")
			return
		}
		created, err := s.control.CreateModel(r.Context(), model)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) modelByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/models/")
	if id == "" || strings.Contains(id, "/") {
		writeErrorStatus(w, http.StatusBadRequest, "model id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		model, err := s.control.GetModel(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, model)
	case http.MethodPut:
		var model domain.Model
		if err := decodeJSON(r, &model); err != nil {
			writeErrorStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		model.ID = id
		if model.Name == "" || model.Provider == "" {
			writeErrorStatus(w, http.StatusBadRequest, "name and provider are required")
			return
		}
		updated, err := s.control.UpdateModel(r.Context(), model)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, updated)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPut)
	}
}

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.control.ListTasks(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req struct {
			Input   string `json:"input"`
			AgentID string `json:"agentId"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeErrorStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.TrimSpace(req.Input) == "" {
			writeErrorStatus(w, http.StatusBadRequest, "input is required")
			return
		}
		task, err := s.control.CreateTask(r.Context(), req.Input, req.AgentID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, task)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) taskResource(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeErrorStatus(w, http.StatusBadRequest, "task id is required")
		return
	}
	taskID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		task, err := s.control.GetTask(r.Context(), taskID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, task)
		return
	}
	if len(parts) != 2 {
		writeErrorStatus(w, http.StatusNotFound, "task resource not found")
		return
	}
	switch parts[1] {
	case "route":
		s.routeTask(w, r, taskID)
	case "routes":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		items, err := s.control.ListRouteDecisions(r.Context(), taskID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case "costs":
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		items, err := s.control.ListCostEvents(r.Context(), taskID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	default:
		writeErrorStatus(w, http.StatusNotFound, "task resource not found")
	}
}

func (s *Server) routeTask(w http.ResponseWriter, r *http.Request, taskID string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req struct {
		RouteClass            domain.RouteClass       `json:"routeClass"`
		RequiredCapabilities []string                `json:"requiredCapabilities,omitempty"`
		InferenceEffort       domain.InferenceEffort  `json:"inferenceEffort,omitempty"`
		ServiceTier           domain.ServiceTier      `json:"serviceTier,omitempty"`
		EstimatedInputTokens  int                     `json:"estimatedInputTokens,omitempty"`
		EstimatedOutputTokens int                     `json:"estimatedOutputTokens,omitempty"`
		Budget                float64                 `json:"budget,omitempty"`
		Currency              string                  `json:"currency,omitempty"`
		DataResidency         string                  `json:"dataResidency,omitempty"`
		EvidenceVersion       string                  `json:"evidenceVersion,omitempty"`
		PolicyVersion         string                  `json:"policyVersion,omitempty"`
		AllowDegraded         bool                    `json:"allowDegraded,omitempty"`
		RequireFreshSignals   bool                    `json:"requireFreshSignals,omitempty"`
		SignalMaxAgeSeconds   int                     `json:"signalMaxAgeSeconds,omitempty"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	decision, err := s.control.DecideRoute(r.Context(), router.Request{
		TaskID:               taskID,
		RouteClass:            req.RouteClass,
		RequiredCapabilities: req.RequiredCapabilities,
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
	})
	if err != nil {
		if strings.Contains(err.Error(), "no policy-compliant model capacity") {
			writeErrorStatus(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, decision)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	writeErrorStatus(w, http.StatusInternalServerError, err.Error())
}

func writeErrorStatus(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeErrorStatus(w, http.StatusMethodNotAllowed, "method not allowed")
}

func requestLogger(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if log != nil {
			log.Info("http request", "method", r.Method, "path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
