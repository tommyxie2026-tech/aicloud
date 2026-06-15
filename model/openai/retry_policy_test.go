package openai

import (
	"errors"
	"net/http"
	"testing"
)

func TestRetryPolicyRetriesTransportError(t *testing.T) {
	policy := NewRetryPolicy(2)
	decision := policy.ShouldRetry(0, 0, errors.New("network error"))
	if !decision.Retry {
		t.Fatalf("expected retry for transport error")
	}
}

func TestRetryPolicyRetriesRetryableStatus(t *testing.T) {
	policy := NewRetryPolicy(2)
	statuses := []int{
		http.StatusTooManyRequests,
		http.StatusRequestTimeout,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}
	for _, status := range statuses {
		decision := policy.ShouldRetry(0, status, nil)
		if !decision.Retry {
			t.Fatalf("expected retry for status %d", status)
		}
	}
}

func TestRetryPolicyDoesNotRetryNonRetryableStatus(t *testing.T) {
	policy := NewRetryPolicy(2)
	statuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}
	for _, status := range statuses {
		decision := policy.ShouldRetry(0, status, nil)
		if decision.Retry {
			t.Fatalf("did not expect retry for status %d", status)
		}
	}
}

func TestRetryPolicyStopsAtMaxRetries(t *testing.T) {
	policy := NewRetryPolicy(2)
	decision := policy.ShouldRetry(2, http.StatusTooManyRequests, nil)
	if decision.Retry {
		t.Fatalf("expected retry to stop at max retries")
	}
}

func TestRetryPolicyNormalizesNegativeRetries(t *testing.T) {
	policy := NewRetryPolicy(-1)
	if policy.MaxRetries != 0 {
		t.Fatalf("expected max retries 0, got %d", policy.MaxRetries)
	}
	decision := policy.ShouldRetry(0, http.StatusTooManyRequests, nil)
	if decision.Retry {
		t.Fatalf("did not expect retry when max retries is 0")
	}
}

func TestIsHTTPClientError(t *testing.T) {
	err := NewHTTPClientError("Non2xxResponse", "status=429")
	if !IsHTTPClientError(err, "Non2xxResponse") {
		t.Fatalf("expected code match")
	}
	if IsHTTPClientError(err, "Other") {
		t.Fatalf("did not expect other code match")
	}
	if IsHTTPClientError(errors.New("plain"), "Non2xxResponse") {
		t.Fatalf("did not expect plain error to match")
	}
}
