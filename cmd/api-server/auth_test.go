package main

import (
	"context"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/httpapi"
)

func TestBuildPrincipalVerifierSelectsTrustedIngressCompatibilityMode(t *testing.T) {
	verifier, err := buildPrincipalVerifier(context.Background(), config.AuthConfig{Mode: config.AuthModeTrustedIngress})
	if err != nil {
		t.Fatalf("buildPrincipalVerifier: %v", err)
	}
	if _, ok := verifier.(httpapi.TrustedIngressVerifier); !ok {
		t.Fatalf("verifier type=%T want TrustedIngressVerifier", verifier)
	}
}

func TestBuildPrincipalVerifierRejectsUnknownMode(t *testing.T) {
	verifier, err := buildPrincipalVerifier(context.Background(), config.AuthConfig{Mode: "unknown"})
	if err == nil {
		t.Fatal("unknown authentication mode was accepted")
	}
	if verifier != nil {
		t.Fatalf("verifier=%T want nil", verifier)
	}
}

func TestBuildPrincipalVerifierFailsClosedOnIncompleteOIDCConfiguration(t *testing.T) {
	verifier, err := buildPrincipalVerifier(context.Background(), config.AuthConfig{Mode: config.AuthModeOIDC})
	if err == nil {
		t.Fatal("incomplete OIDC configuration was accepted")
	}
	if verifier != nil {
		t.Fatalf("verifier=%T want nil", verifier)
	}
}
