package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/admission"
	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/evaluation"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func (s *Server) registerEvidenceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/models/{id}/admission", s.modelAdmission)
	mux.HandleFunc("POST /api/v1/models/{id}/admission", s.modelAdmission)
	mux.HandleFunc("POST /api/v1/tasks/{id}/model", s.executeModel)
	mux.HandleFunc("GET /api/v1/tasks/{id}/trace", s.taskTrace)
	mux.HandleFunc("GET /api/v1/tasks/{id}/evaluations", s.taskEvaluations)
	mux.HandleFunc("POST /api/v1/tasks/{id}/evaluations", s.taskEvaluations)
}

func (s *Server) modelAdmission(w http.ResponseWriter, r *http.Request) {
	modelID := r.PathValue("id")
	if modelID == "" {
		writeErrorStatus(w, http.StatusBadRequest, "model id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		evidence, err := s.control.ListAdmissionEvidence(r.Context(), modelID)
		if err != nil {
			writeEvidenceError(w, err)
			return
		}
		decision, err := s.control.CheckModelAdmission(r.Context(), modelID)
		if err != nil {
			writeEvidenceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"decision": decision, "evidence": evidence})
	case http.MethodPost:
		var evidence admission.Evidence
		if err := decodeJSON(r, &evidence); err != nil {
			writeErrorStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		created, err := s.control.AppendAdmissionEvidence(r.Context(), modelID, evidence)
		if err != nil {
			writeEvidenceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) executeModel(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeErrorStatus(w, http.StatusBadRequest, "task id is required")
		return
	}
	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" {
		writeErrorStatus(w, http.StatusBadRequest, "Idempotency-Key is required")
		return
	}
	var request provider.ProviderRequest
	if err := decodeJSON(r, &request); err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	businessRequest := request
	businessRequest.RequestID = ""
	digest, err := canonicalRequestDigest(struct {
		TaskID  string                   `json:"taskId"`
		Request provider.ProviderRequest `json:"request"`
	}{TaskID: taskID, Request: businessRequest})
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.control.ExecuteModelIdempotent(r.Context(), taskID, request, controlplane.CommandMetadata{
		IdempotencyKey: key,
		RequestDigest:  digest,
		RequestID:      strings.TrimSpace(r.Header.Get("X-Request-ID")),
	})
	if result.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrIdempotencyConflict):
			writeErrorStatus(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT")
		case errors.Is(err, repository.ErrIdempotencyInProgress):
			writeErrorStatus(w, http.StatusConflict, "IDEMPOTENCY_IN_PROGRESS")
		case errors.Is(err, domain.ErrInvalidTaskTransition):
			writeErrorStatus(w, http.StatusConflict, err.Error())
		default:
			writeModelExecutionError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, result.Result)
}

func (s *Server) taskTrace(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	items, err := s.control.ListTrace(r.Context(), taskID)
	if err != nil {
		writeEvidenceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) taskEvaluations(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeErrorStatus(w, http.StatusBadRequest, "task id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.control.ListEvaluations(r.Context(), taskID)
		if err != nil {
			writeEvidenceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var request struct {
			Config     evaluation.Config     `json:"config"`
			Metrics    evaluation.Metrics    `json:"metrics"`
			Thresholds evaluation.Thresholds `json:"thresholds"`
			RawOutput  string                `json:"rawOutput,omitempty"`
		}
		if err := decodeJSON(r, &request); err != nil {
			writeErrorStatus(w, http.StatusBadRequest, err.Error())
			return
		}
		run, err := s.control.CreateEvaluation(
			r.Context(), taskID, request.Config, request.Metrics,
			request.Thresholds, []byte(request.RawOutput),
		)
		if err != nil {
			writeEvidenceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, run)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func writeEvidenceError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) || strings.Contains(err.Error(), "not found") {
		writeErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "already exists") {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	writeErrorStatus(w, http.StatusInternalServerError, err.Error())
}

func writeModelExecutionError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) || strings.Contains(err.Error(), "not found") {
		writeErrorStatus(w, http.StatusNotFound, err.Error())
		return
	}
	if strings.Contains(err.Error(), "does not have a route decision") {
		writeErrorStatus(w, http.StatusConflict, err.Error())
		return
	}
	var providerErr *provider.ProviderError
	if errors.As(err, &providerErr) && providerErr.Retryable {
		writeErrorStatus(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeErrorStatus(w, http.StatusInternalServerError, err.Error())
}
