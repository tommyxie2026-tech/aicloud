package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOIDCVerifierTreatsScalarAudienceAsOneExactValue(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	token := signRS256(t, privateKey, "key-1", map[string]any{
		"iss":       issuer,
		"sub":       "user-123",
		"aud":       "other aicloud-api",
		"exp":       time.Now().Add(5 * time.Minute).Unix(),
		"tenant_id": "tenant-a",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	if _, err := verifier.Verify(request); err == nil {
		t.Fatal("packed scalar audience was accepted")
	}
}

func TestOIDCVerifierRejectsUnsignedAlgorithm(t *testing.T) {
	issuer, client, _ := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer eyJhbGciOiJub25lIiwia2lkIjoia2V5LTEifQ.eyJpc3MiOiJ4In0.signature")
	if _, err := verifier.Verify(request); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}
