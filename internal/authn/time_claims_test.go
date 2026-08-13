package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOIDCVerifierRequiresExpiration(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	token := signRS256(t, privateKey, "key-1", map[string]any{
		"iss":       issuer,
		"sub":       "user-123",
		"aud":       "aicloud-api",
		"tenant_id": "tenant-a",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	if _, err := verifier.Verify(request); err == nil {
		t.Fatal("token without exp was accepted")
	}
}

func TestOIDCVerifierRejectsNotYetValidTokenBeyondClockSkew(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api", ClockSkew: 5 * time.Second}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	now := time.Now()
	token := signRS256(t, privateKey, "key-1", map[string]any{
		"iss":       issuer,
		"sub":       "user-123",
		"aud":       "aicloud-api",
		"exp":       now.Add(10 * time.Minute).Unix(),
		"nbf":       now.Add(time.Minute).Unix(),
		"tenant_id": "tenant-a",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	if _, err := verifier.Verify(request); err != ErrTokenNotYetValid {
		t.Fatalf("not-yet-valid token error=%v want=%v", err, ErrTokenNotYetValid)
	}
}
