package httpapi

import "net/http"

type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	TraceID   string         `json:"trace_id"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, retryable bool, details map[string]any) {
	requestID := normalizeCorrelationID(w.Header().Get(RequestIDHeader))
	if requestID == "" {
		requestID = newCorrelationID("req")
		w.Header().Set(RequestIDHeader, requestID)
	}
	traceID := normalizeCorrelationID(w.Header().Get(TraceIDHeader))
	if traceID == "" {
		traceID = newCorrelationID("trace")
		w.Header().Set(TraceIDHeader, traceID)
	}
	writeJSON(w, status, ErrorEnvelope{Error: APIError{
		Code: code, Message: message, RequestID: requestID, TraceID: traceID,
		Retryable: retryable, Details: details,
	}})
}

func defaultAPIErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "INVALID_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		if status >= 500 {
			return "INTERNAL_ERROR"
		}
		return "REQUEST_FAILED"
	}
}

func defaultRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status >= 500
}
