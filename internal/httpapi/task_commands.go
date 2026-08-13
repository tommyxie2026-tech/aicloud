package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/controlplane"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

const idempotencyKeyHeader = "Idempotency-Key"

func (s *Server) taskCollectionCommandAware(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.tasks(w, r)
		return
	}

	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" {
		writeErrorStatus(w, http.StatusBadRequest, "Idempotency-Key is required")
		return
	}
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
	digest, err := canonicalRequestDigest(req)
	if err != nil {
		writeErrorStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.control.CreateTaskIdempotent(r.Context(), req.Input, req.AgentID, controlplane.CommandMetadata{
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
		default:
			writeError(w, err)
		}
		return
	}
	if result.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(w, http.StatusAccepted, result.Task)
}

func canonicalRequestDigest(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
