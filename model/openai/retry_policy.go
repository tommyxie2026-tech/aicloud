package openai

import (
	"errors"
	"net/http"
)

type RetryPolicy struct {
	MaxRetries int
}

type RetryDecision struct {
	Retry  bool
	Reason string
}

func NewRetryPolicy(maxRetries int) RetryPolicy {
	if maxRetries < 0 {
		maxRetries = 0
	}
	return RetryPolicy{MaxRetries: maxRetries}
}

func (p RetryPolicy) ShouldRetry(attempt int, statusCode int, err error) RetryDecision {
	if attempt >= p.MaxRetries {
		return RetryDecision{Retry: false, Reason: "max retries reached"}
	}
	if err != nil {
		return RetryDecision{Retry: true, Reason: "transport error"}
	}
	if isRetryableStatus(statusCode) {
		return RetryDecision{Retry: true, Reason: "retryable status"}
	}
	return RetryDecision{Retry: false, Reason: "not retryable"}
}

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusRequestTimeout,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func IsHTTPClientError(err error, code string) bool {
	var clientErr *HTTPClientError
	if errors.As(err, &clientErr) {
		return clientErr.Code == code
	}
	return false
}
