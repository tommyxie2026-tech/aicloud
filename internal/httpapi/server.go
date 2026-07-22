package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"log/slog"
	"net/http"
	"strings"
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
	mux.HandleFunc("/api/v1/tasks", s.tasks)
	mux.HandleFunc("/api/v1/tasks/", s.taskByID)
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
func (s *Server) taskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	if id == "" {
		writeErrorStatus(w, http.StatusBadRequest, "task id is required")
		return
	}
	task, err := s.control.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
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
