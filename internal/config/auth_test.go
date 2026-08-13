package config

import (
	"reflect"
	"testing"
)

func TestLoadAuthConfigDefaultsToExplicitTrustedIngressCompatibility(t *testing.T) {
	t.Setenv("AICLOUD_AUTH_MODE", "")
	t.Setenv("AICLOUD_OIDC_ALLOWED_ALGORITHMS", "")
	cfg := loadAuthConfig()
	if cfg.Mode != AuthModeTrustedIngress {
		t.Fatalf("mode=%q want=%q", cfg.Mode, AuthModeTrustedIngress)
	}
	if !reflect.DeepEqual(cfg.AllowedAlgorithms, []string{"RS256"}) {
		t.Fatalf("algorithms=%v want=[RS256]", cfg.AllowedAlgorithms)
	}
	if cfg.TenantClaim != "tenant_id" || cfg.ProjectClaim != "project_id" {
		t.Fatalf("unexpected scope claim defaults: tenant=%q project=%q", cfg.TenantClaim, cfg.ProjectClaim)
	}
	if cfg.ClockSkewSeconds != 60 || cfg.JWKSCacheTTLSeconds != 300 {
		t.Fatalf("unexpected timing defaults: skew=%d cache=%d", cfg.ClockSkewSeconds, cfg.JWKSCacheTTLSeconds)
	}
}

func TestLoadAuthConfigReadsOIDCSettings(t *testing.T) {
	t.Setenv("AICLOUD_AUTH_MODE", "OIDC")
	t.Setenv("AICLOUD_OIDC_ISSUER", "https://id.example.com")
	t.Setenv("AICLOUD_OIDC_AUDIENCE", "aicloud-api")
	t.Setenv("AICLOUD_OIDC_JWKS_URL", "https://id.example.com/keys")
	t.Setenv("AICLOUD_OIDC_ALLOWED_ALGORITHMS", "RS256, RS512,RS256")
	t.Setenv("AICLOUD_OIDC_TENANT_CLAIM", "org_id")
	t.Setenv("AICLOUD_OIDC_PROJECT_CLAIM", "workspace_id")
	t.Setenv("AICLOUD_OIDC_CLOCK_SKEW_SECONDS", "30")
	t.Setenv("AICLOUD_OIDC_JWKS_CACHE_TTL_SECONDS", "600")

	cfg := loadAuthConfig()
	if cfg.Mode != AuthModeOIDC || cfg.Issuer != "https://id.example.com" || cfg.Audience != "aicloud-api" || cfg.JWKSURL != "https://id.example.com/keys" {
		t.Fatalf("unexpected OIDC config: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.AllowedAlgorithms, []string{"RS256", "RS512"}) {
		t.Fatalf("algorithms=%v", cfg.AllowedAlgorithms)
	}
	if cfg.TenantClaim != "org_id" || cfg.ProjectClaim != "workspace_id" || cfg.ClockSkewSeconds != 30 || cfg.JWKSCacheTTLSeconds != 600 {
		t.Fatalf("unexpected mapped OIDC settings: %+v", cfg)
	}
}
