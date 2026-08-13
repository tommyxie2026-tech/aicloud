package authn

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestOIDCVerifierMapsVerifiedClaims(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	now := time.Now().UTC()
	token := signRS256(t, privateKey, "key-1", map[string]any{
		"iss":            issuer,
		"sub":            "user-123",
		"aud":            []string{"other", "aicloud-api"},
		"exp":            now.Add(5 * time.Minute).Unix(),
		"iat":            now.Add(-time.Minute).Unix(),
		"tenant_id":      "tenant-a",
		"project_id":     "project-a",
		"roles":          []string{"developer", "reviewer"},
		"capabilities":   "task:read task:create",
		"principal_type": "user",
		"sid":            "session-1",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	principal, err := verifier.Verify(request)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if principal.Type != identity.PrincipalUser || principal.SubjectID != "user-123" || principal.TenantID != "tenant-a" || principal.ProjectID != "project-a" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
	if principal.AuthnMethod != "oidc_jwt" || principal.Issuer != issuer || principal.SessionID != "session-1" {
		t.Fatalf("unexpected authentication evidence: %+v", principal)
	}
	if !principal.HasCapability("task:create") {
		t.Fatalf("capabilities not mapped: %+v", principal.Capabilities)
	}
}

func TestOIDCVerifierRejectsWrongAudienceAndExpiredToken(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api", ClockSkew: time.Second}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	baseClaims := map[string]any{
		"iss":       issuer,
		"sub":       "user-123",
		"tenant_id": "tenant-a",
		"exp":       time.Now().Add(5 * time.Minute).Unix(),
	}

	wrongAudience := cloneClaims(baseClaims)
	wrongAudience["aud"] = "another-api"
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+signRS256(t, privateKey, "key-1", wrongAudience))
	if _, err := verifier.Verify(request); err == nil {
		t.Fatal("wrong audience token was accepted")
	}

	expired := cloneClaims(baseClaims)
	expired["aud"] = "aicloud-api"
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	request = httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+signRS256(t, privateKey, "key-1", expired))
	if _, err := verifier.Verify(request); err != ErrTokenExpired {
		t.Fatalf("expired token error=%v want=%v", err, ErrTokenExpired)
	}
}

func TestOIDCVerifierRejectsExternalSystemPrincipal(t *testing.T) {
	issuer, client, privateKey := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	token := signRS256(t, privateKey, "key-1", map[string]any{
		"iss":            issuer,
		"sub":            "system-claim",
		"aud":            "aicloud-api",
		"exp":            time.Now().Add(5 * time.Minute).Unix(),
		"tenant_id":      "tenant-a",
		"principal_type": "system",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	if _, err := verifier.Verify(request); err == nil {
		t.Fatal("external system principal was accepted")
	}
}

func TestOIDCVerifierRefreshesJWKSOnKeyRotation(t *testing.T) {
	key1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	key2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	activeKid := "key-1"
	activeKey := key1
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": server.URL, "jwks_uri": server.URL + "/jwks"})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{publicJWK(activeKid, &activeKey.PublicKey)}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: server.URL, Audience: "aicloud-api", JWKSCacheTTL: time.Hour}, server.Client())
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}

	activeKid = "key-2"
	activeKey = key2
	token := signRS256(t, key2, "key-2", map[string]any{
		"iss":       server.URL,
		"sub":       "svc-1",
		"aud":       "aicloud-api",
		"exp":       time.Now().Add(5 * time.Minute).Unix(),
		"tenant_id": "tenant-a",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	if _, err := verifier.Verify(request); err != nil {
		t.Fatalf("rotated key token rejected: %v", err)
	}
}

func TestOIDCVerifierRequiresSingleBearerHeader(t *testing.T) {
	issuer, client, _ := oidcTestServer(t, "key-1")
	verifier, err := NewOIDCVerifier(context.Background(), OIDCConfig{Issuer: issuer, Audience: "aicloud-api"}, client)
	if err != nil {
		t.Fatalf("NewOIDCVerifier: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	if _, err := verifier.Verify(request); err != ErrBearerRequired {
		t.Fatalf("missing bearer error=%v want=%v", err, ErrBearerRequired)
	}
	request.Header.Add("Authorization", "Bearer one")
	request.Header.Add("Authorization", "Bearer two")
	if _, err := verifier.Verify(request); err != ErrBearerRequired {
		t.Fatalf("duplicate bearer error=%v want=%v", err, ErrBearerRequired)
	}
}

func oidcTestServer(t *testing.T, kid string) (string, *http.Client, *rsa.PrivateKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": server.URL, "jwks_uri": server.URL + "/jwks"})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{publicJWK(kid, &privateKey.PublicKey)}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, server.Client(), privateKey
}

func publicJWK(kid string, key *rsa.PublicKey) map[string]any {
	exponent := big.NewInt(int64(key.E)).Bytes()
	return map[string]any{
		"kty": "RSA",
		"kid": kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": kid, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	headerPart := base64.RawURLEncoding.EncodeToString(header)
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	input := headerPart + "." + payloadPart
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%s.%s", input, base64.RawURLEncoding.EncodeToString(signature))
}

func cloneClaims(source map[string]any) map[string]any {
	copy := make(map[string]any, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
