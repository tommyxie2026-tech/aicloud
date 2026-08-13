package config

import "strings"

const (
	AuthModeTrustedIngress = "trusted_ingress"
	AuthModeOIDC           = "oidc"
)

// AuthConfig selects the API authentication adapter and defines the OIDC
// resource-server verification contract. Trusted ingress remains an explicit
// compatibility mode; it is never inferred from missing configuration at the
// HTTP boundary.
type AuthConfig struct {
	Mode                string
	Issuer              string
	Audience            string
	JWKSURL             string
	AllowedAlgorithms   []string
	TenantClaim         string
	ProjectClaim        string
	RolesClaim          string
	CapabilitiesClaim   string
	PrincipalTypeClaim  string
	SessionClaim        string
	ClockSkewSeconds    int
	JWKSCacheTTLSeconds int
}

func loadAuthConfig() AuthConfig {
	return AuthConfig{
		Mode:                strings.ToLower(strings.TrimSpace(env("AICLOUD_AUTH_MODE", AuthModeTrustedIngress))),
		Issuer:              strings.TrimSpace(env("AICLOUD_OIDC_ISSUER", "")),
		Audience:            strings.TrimSpace(env("AICLOUD_OIDC_AUDIENCE", "")),
		JWKSURL:             strings.TrimSpace(env("AICLOUD_OIDC_JWKS_URL", "")),
		AllowedAlgorithms:   splitCSV(env("AICLOUD_OIDC_ALLOWED_ALGORITHMS", "RS256")),
		TenantClaim:         strings.TrimSpace(env("AICLOUD_OIDC_TENANT_CLAIM", "tenant_id")),
		ProjectClaim:        strings.TrimSpace(env("AICLOUD_OIDC_PROJECT_CLAIM", "project_id")),
		RolesClaim:          strings.TrimSpace(env("AICLOUD_OIDC_ROLES_CLAIM", "roles")),
		CapabilitiesClaim:   strings.TrimSpace(env("AICLOUD_OIDC_CAPABILITIES_CLAIM", "capabilities")),
		PrincipalTypeClaim:  strings.TrimSpace(env("AICLOUD_OIDC_PRINCIPAL_TYPE_CLAIM", "principal_type")),
		SessionClaim:        strings.TrimSpace(env("AICLOUD_OIDC_SESSION_CLAIM", "sid")),
		ClockSkewSeconds:    envInt("AICLOUD_OIDC_CLOCK_SKEW_SECONDS", 60),
		JWKSCacheTTLSeconds: envInt("AICLOUD_OIDC_JWKS_CACHE_TTL_SECONDS", 300),
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		items = append(items, part)
	}
	return items
}
