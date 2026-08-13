package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/authn"
	"github.com/tommyxie2026-tech/aicloud/internal/config"
	"github.com/tommyxie2026-tech/aicloud/internal/httpapi"
)

func buildPrincipalVerifier(ctx context.Context, cfg config.AuthConfig) (httpapi.PrincipalVerifier, error) {
	switch cfg.Mode {
	case config.AuthModeTrustedIngress:
		return httpapi.TrustedIngressVerifier{}, nil
	case config.AuthModeOIDC:
		return authn.NewOIDCVerifier(ctx, authn.OIDCConfig{
			Issuer:             cfg.Issuer,
			Audience:           cfg.Audience,
			JWKSURL:            cfg.JWKSURL,
			AllowedAlgorithms:  cfg.AllowedAlgorithms,
			TenantClaim:        cfg.TenantClaim,
			ProjectClaim:       cfg.ProjectClaim,
			RolesClaim:         cfg.RolesClaim,
			CapabilitiesClaim:  cfg.CapabilitiesClaim,
			PrincipalTypeClaim: cfg.PrincipalTypeClaim,
			SessionClaim:       cfg.SessionClaim,
			ClockSkew:          time.Duration(cfg.ClockSkewSeconds) * time.Second,
			JWKSCacheTTL:       time.Duration(cfg.JWKSCacheTTLSeconds) * time.Second,
		}, nil)
	default:
		return nil, fmt.Errorf("unsupported authentication mode %q", cfg.Mode)
	}
}
