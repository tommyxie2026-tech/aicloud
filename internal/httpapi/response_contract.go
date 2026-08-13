package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	DefaultPageSize       = 50
	MaxPageSize           = 200
	ResourceVersionHeader = "X-Resource-Version"
)

var errPageOffset = errors.New("page offset exceeds collection size")

type bufferedResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
}

func (w *bufferedResponseWriter) Write(payload []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(payload)
}

func WithAPIResponseContract(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pagination, paginated, err := paginationRequest(r)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), false, nil)
			return
		}

		buffered := newBufferedResponseWriter()
		next.ServeHTTP(buffered, r)
		status := buffered.status
		body := buffered.body.Bytes()
		mergeHeaders(w.Header(), buffered.header)

		if status >= 400 {
			if envelope, ok := normalizeLegacyError(status, body); ok {
				writeAPIError(w, status, envelope.Code, envelope.Message, envelope.Retryable, envelope.Details)
				return
			}
		}

		if paginated && status >= 200 && status < 300 {
			pagedBody, err := paginateJSON(body, pagination)
			if err != nil {
				if errors.Is(err, errPageOffset) {
					writeAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "pageToken does not identify a valid page", false, nil)
					return
				}
				writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "list response does not satisfy pagination contract", false, nil)
				return
			}
			body = pagedBody
		}

		if isDirectTaskRead(r) && status == http.StatusOK {
			setTaskResourceVersionHeaders(w.Header(), body)
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

type paginationSpec struct {
	PageSize int
	Offset   int
}

func paginationRequest(r *http.Request) (paginationSpec, bool, error) {
	if r == nil || r.Method != http.MethodGet || !isPaginatedPath(r.URL.Path) {
		return paginationSpec{}, false, nil
	}

	spec := paginationSpec{PageSize: DefaultPageSize}
	if value := strings.TrimSpace(r.URL.Query().Get("pageSize")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > MaxPageSize {
			return paginationSpec{}, true, fmt.Errorf("pageSize must be between 1 and %d", MaxPageSize)
		}
		spec.PageSize = parsed
	}
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		offset, err := decodePageToken(token)
		if err != nil {
			return paginationSpec{}, true, fmt.Errorf("pageToken is invalid")
		}
		spec.Offset = offset
	}
	return spec, true, nil
}

func isPaginatedPath(path string) bool {
	if path == "/api/v1/models" || path == "/api/v1/tools" || path == "/api/v1/tasks" {
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "tasks" || parts[3] == "" {
		return false
	}
	switch parts[4] {
	case "routes", "costs", "audit", "trace", "evaluations":
		return true
	default:
		return false
	}
}

func paginateJSON(body []byte, spec paginationSpec) ([]byte, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	if spec.Offset > len(items) {
		return nil, errPageOffset
	}

	end := spec.Offset + spec.PageSize
	if end > len(items) {
		end = len(items)
	}
	page := items[spec.Offset:end]
	if page == nil {
		page = []json.RawMessage{}
	}
	response := struct {
		Items         []json.RawMessage `json:"items"`
		NextPageToken string            `json:"nextPageToken,omitempty"`
	}{Items: page}
	if end < len(items) {
		response.NextPageToken = encodePageToken(end)
	}
	return json.Marshal(response)
}

func encodePageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("v1:%d", offset)))
}

func decodePageToken(token string) (int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, err
	}
	value := string(decoded)
	if !strings.HasPrefix(value, "v1:") {
		return 0, fmt.Errorf("unsupported token version")
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(value, "v1:"))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid offset")
	}
	return offset, nil
}

func isDirectTaskRead(r *http.Request) bool {
	if r == nil || r.Method != http.MethodGet {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	return len(parts) == 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "tasks" && parts[3] != ""
}

func setTaskResourceVersionHeaders(header http.Header, body []byte) {
	var task struct {
		ID      string `json:"id"`
		Version int64  `json:"version"`
	}
	if err := json.Unmarshal(body, &task); err != nil || task.ID == "" || task.Version < 1 {
		return
	}
	header.Set(ResourceVersionHeader, strconv.FormatInt(task.Version, 10))
	header.Set("ETag", fmt.Sprintf("\"task:%s:v%d\"", task.ID, task.Version))
}

func mergeHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

func normalizeLegacyError(status int, body []byte) (APIError, bool) {
	var raw struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || len(raw.Error) == 0 {
		return APIError{}, false
	}
	if raw.Error[0] == '{' {
		return APIError{}, false
	}

	var message string
	if err := json.Unmarshal(raw.Error, &message); err != nil {
		return APIError{}, false
	}
	code := defaultAPIErrorCode(status)
	switch message {
	case "IDEMPOTENCY_CONFLICT":
		code = "IDEMPOTENCY_CONFLICT"
	case "IDEMPOTENCY_IN_PROGRESS":
		code = "IDEMPOTENCY_IN_PROGRESS"
	}
	return APIError{Code: code, Message: message, Retryable: defaultRetryable(status)}, true
}
