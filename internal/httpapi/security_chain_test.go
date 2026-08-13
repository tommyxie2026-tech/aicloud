package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/authorization"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type failingVerifier struct{}

func (failingVerifier) Verify(*http.Request) (identity.Principal, error) {
	return identity.Principal{}, errors.New("verification failed")
}

type callCountingAuthorizer struct{ calls int }

func (a *callCountingAuthorizer) Authorize(context.Context, authorization.Request) (authorization.Decision, error) {
	a.calls++
	return authorization.Decision{Allowed: true}, nil
}

func TestVerifierRunsBeforeAPIAccessEvaluation(t *testing.T) {
	authorizer := &callCountingAuthorizer{}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("business handler unexpectedly called")
	})
	handler := WithRequestMetadata(WithPrincipalVerifier(failingVerifier{}, WithAuthorization(authorizer, next)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
	if authorizer.calls != 0 {
		t.Fatalf("access evaluator called %d times", authorizer.calls)
	}
}
