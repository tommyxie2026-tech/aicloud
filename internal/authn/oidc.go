package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

const (
	defaultHTTPTimeout = 5 * time.Second
	defaultJWKSCache   = 5 * time.Minute
	defaultClockSkew   = 60 * time.Second
	maxTokenBytes      = 16 * 1024
	maxRedirects       = 10
)

var (
	ErrBearerRequired   = errors.New("bearer token is required")
	ErrTokenInvalid     = errors.New("token is invalid")
	ErrTokenExpired     = errors.New("token is expired")
	ErrTokenNotYetValid = errors.New("token is not yet valid")
)

// OIDCConfig defines the fail-closed production authentication contract.
// Issuer and Audience are always explicit. JWKSURL may be omitted when the
// issuer supports OpenID Connect discovery.
type OIDCConfig struct {
	Issuer             string
	Audience           string
	JWKSURL            string
	AllowedAlgorithms  []string
	TenantClaim        string
	ProjectClaim       string
	RolesClaim         string
	CapabilitiesClaim  string
	PrincipalTypeClaim string
	SessionClaim       string
	ClockSkew          time.Duration
	JWKSCacheTTL       time.Duration
}

// OIDCVerifier verifies bearer JWTs cryptographically and maps only verified
// claims into the platform Principal. External tokens can never assert a
// System principal.
type OIDCVerifier struct {
	cfg  OIDCConfig
	keys *remoteKeySet
	now  func() time.Time
}

type discoveryDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

type jwtHeader struct {
	Alg  string   `json:"alg"`
	Kid  string   `json:"kid"`
	Typ  string   `json:"typ"`
	Crit []string `json:"crit"`
}

func NewOIDCVerifier(ctx context.Context, cfg OIDCConfig, client *http.Client) (*OIDCVerifier, error) {
	cfg = normalizeConfig(cfg)
	if err := validateHTTPSURL("issuer", cfg.Issuer); err != nil {
		return nil, err
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("audience is required")
	}
	if len(cfg.AllowedAlgorithms) == 0 {
		cfg.AllowedAlgorithms = []string{"RS256"}
	}
	for _, alg := range cfg.AllowedAlgorithms {
		if !supportedAlgorithm(alg) {
			return nil, fmt.Errorf("unsupported JWT signing algorithm %q", alg)
		}
	}

	httpClient := cloneHTTPClient(client)
	jwksURL := cfg.JWKSURL
	if jwksURL == "" {
		discovered, err := discover(ctx, httpClient, cfg.Issuer)
		if err != nil {
			return nil, err
		}
		if discovered.Issuer != cfg.Issuer {
			return nil, fmt.Errorf("OIDC discovery issuer mismatch")
		}
		jwksURL = discovered.JWKSURI
	}
	if err := validateHTTPSURL("jwks_url", jwksURL); err != nil {
		return nil, err
	}

	keys := newRemoteKeySet(httpClient, jwksURL, cfg.JWKSCacheTTL, cfg.AllowedAlgorithms)
	if err := keys.refresh(ctx); err != nil {
		return nil, err
	}
	return &OIDCVerifier{cfg: cfg, keys: keys, now: time.Now}, nil
}

func normalizeConfig(cfg OIDCConfig) OIDCConfig {
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.Audience = strings.TrimSpace(cfg.Audience)
	cfg.JWKSURL = strings.TrimSpace(cfg.JWKSURL)
	cfg.AllowedAlgorithms = normalizeStrings(cfg.AllowedAlgorithms)
	cfg.TenantClaim = defaultClaimName(cfg.TenantClaim, "tenant_id")
	cfg.ProjectClaim = defaultClaimName(cfg.ProjectClaim, "project_id")
	cfg.RolesClaim = defaultClaimName(cfg.RolesClaim, "roles")
	cfg.CapabilitiesClaim = defaultClaimName(cfg.CapabilitiesClaim, "capabilities")
	cfg.PrincipalTypeClaim = defaultClaimName(cfg.PrincipalTypeClaim, "principal_type")
	cfg.SessionClaim = defaultClaimName(cfg.SessionClaim, "sid")
	if cfg.ClockSkew <= 0 {
		cfg.ClockSkew = defaultClockSkew
	}
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = defaultJWKSCache
	}
	return cfg
}

func defaultClaimName(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func cloneHTTPClient(client *http.Client) *http.Client {
	var copy http.Client
	if client != nil {
		copy = *client
	}
	if copy.Timeout <= 0 {
		copy.Timeout = defaultHTTPTimeout
	}
	previousRedirectPolicy := copy.CheckRedirect
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req == nil || req.URL == nil || req.URL.Scheme != "https" {
			return fmt.Errorf("OIDC metadata redirect must remain HTTPS")
		}
		if len(via) >= maxRedirects {
			return fmt.Errorf("OIDC metadata redirect limit exceeded")
		}
		if previousRedirectPolicy != nil {
			return previousRedirectPolicy(req, via)
		}
		return nil
	}
	return &copy
}

func validateHTTPSURL(name, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%s must be an absolute HTTPS URL", name)
	}
	return nil
}

func discover(ctx context.Context, client *http.Client, issuer string) (discoveryDocument, error) {
	endpoint := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	var document discoveryDocument
	if err := fetchJSON(ctx, client, endpoint, &document); err != nil {
		return document, fmt.Errorf("OIDC discovery failed: %w", err)
	}
	if document.Issuer == "" || document.JWKSURI == "" {
		return document, fmt.Errorf("OIDC discovery response is incomplete")
	}
	return document, nil
}

// Verify implements the HTTP API PrincipalVerifier contract without importing
// the HTTP API package.
func (v *OIDCVerifier) Verify(r *http.Request) (identity.Principal, error) {
	if r == nil {
		return identity.Principal{}, ErrBearerRequired
	}
	raw, err := bearerToken(r.Header.Values("Authorization"))
	if err != nil {
		return identity.Principal{}, err
	}
	return v.verifyToken(r.Context(), raw)
}

func bearerToken(values []string) (string, error) {
	if len(values) != 1 {
		return "", ErrBearerRequired
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", ErrBearerRequired
	}
	if len(parts[1]) > maxTokenBytes {
		return "", fmt.Errorf("%w: bearer token is too large", ErrTokenInvalid)
	}
	return parts[1], nil
}

func (v *OIDCVerifier) verifyToken(ctx context.Context, raw string) (identity.Principal, error) {
	segments := strings.Split(raw, ".")
	if len(segments) != 3 || segments[0] == "" || segments[1] == "" || segments[2] == "" {
		return identity.Principal{}, ErrTokenInvalid
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		return identity.Principal{}, ErrTokenInvalid
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return identity.Principal{}, ErrTokenInvalid
	}
	if header.Kid == "" || len(header.Crit) != 0 || !contains(v.cfg.AllowedAlgorithms, header.Alg) {
		return identity.Principal{}, ErrTokenInvalid
	}
	if err := v.keys.verify(ctx, header.Alg, header.Kid, []byte(segments[0]+"."+segments[1]), segments[2]); err != nil {
		return identity.Principal{}, ErrTokenInvalid
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		return identity.Principal{}, ErrTokenInvalid
	}
	claims, err := decodeClaims(payloadBytes)
	if err != nil {
		return identity.Principal{}, ErrTokenInvalid
	}
	if err := v.validateClaims(claims); err != nil {
		return identity.Principal{}, err
	}
	return v.principalFromClaims(claims)
}

func decodeClaims(payload []byte) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	var claims map[string]any
	if err := decoder.Decode(&claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (v *OIDCVerifier) validateClaims(claims map[string]any) error {
	issuer, ok := claimString(claims, "iss")
	if !ok || issuer != v.cfg.Issuer {
		return ErrTokenInvalid
	}
	audiences, ok := claimAudienceList(claims["aud"])
	if !ok || !contains(audiences, v.cfg.Audience) {
		return ErrTokenInvalid
	}
	if subject, ok := claimString(claims, "sub"); !ok || subject == "" {
		return ErrTokenInvalid
	}

	now := v.now()
	expiry, ok, err := numericDate(claims["exp"])
	if err != nil || !ok {
		return ErrTokenInvalid
	}
	if now.After(expiry.Add(v.cfg.ClockSkew)) {
		return ErrTokenExpired
	}
	if nbf, exists, err := numericDate(claims["nbf"]); err != nil {
		return ErrTokenInvalid
	} else if exists && now.Add(v.cfg.ClockSkew).Before(nbf) {
		return ErrTokenNotYetValid
	}
	if iat, exists, err := numericDate(claims["iat"]); err != nil {
		return ErrTokenInvalid
	} else if exists {
		if iat.After(now.Add(v.cfg.ClockSkew)) {
			return ErrTokenNotYetValid
		}
		if expiry.Before(iat) {
			return ErrTokenInvalid
		}
	}
	return nil
}

func numericDate(value any) (time.Time, bool, error) {
	if value == nil {
		return time.Time{}, false, nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return time.Time{}, true, fmt.Errorf("numeric date must be a JSON number")
	}
	seconds, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return time.Time{}, true, fmt.Errorf("invalid numeric date")
	}
	whole, fractional := math.Modf(seconds)
	return time.Unix(int64(whole), int64(fractional*float64(time.Second))).UTC(), true, nil
}

func (v *OIDCVerifier) principalFromClaims(claims map[string]any) (identity.Principal, error) {
	subject, _ := claimString(claims, "sub")
	tenant, _ := claimString(claims, v.cfg.TenantClaim)
	project, _ := claimString(claims, v.cfg.ProjectClaim)
	principalType := identity.PrincipalUser
	if value, ok := claimString(claims, v.cfg.PrincipalTypeClaim); ok && value != "" {
		principalType = identity.PrincipalType(value)
	}
	if principalType == identity.PrincipalSystem {
		return identity.Principal{}, fmt.Errorf("external JWT cannot assert a system principal")
	}
	if principalType != identity.PrincipalUser && principalType != identity.PrincipalServiceAccount {
		return identity.Principal{}, ErrTokenInvalid
	}
	roles, _ := claimStringList(claims[v.cfg.RolesClaim])
	capabilities, _ := claimStringList(claims[v.cfg.CapabilitiesClaim])
	sessionID, _ := claimString(claims, v.cfg.SessionClaim)
	principal := identity.Principal{
		Type:         principalType,
		SubjectID:    subject,
		TenantID:     tenant,
		ProjectID:    project,
		Roles:        roles,
		Capabilities: capabilities,
		AuthnMethod:  "oidc_jwt",
		Issuer:       v.cfg.Issuer,
		SessionID:    sessionID,
	}
	if err := identity.Validate(principal); err != nil {
		return identity.Principal{}, fmt.Errorf("verified JWT principal is invalid: %w", err)
	}
	return principal, nil
}

func claimString(claims map[string]any, name string) (string, bool) {
	value, ok := claims[name]
	if !ok {
		return "", false
	}
	stringValue, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(stringValue), true
}

// claimAudienceList follows JWT audience semantics: a scalar aud value is one
// exact audience, not a whitespace-delimited list.
func claimAudienceList(value any) ([]string, bool) {
	switch typed := value.(type) {
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return nil, false
		}
		return []string{typed}, true
	case []any:
		items := make([]string, 0, len(typed))
		for _, entry := range typed {
			stringValue, ok := entry.(string)
			if !ok {
				return nil, false
			}
			stringValue = strings.TrimSpace(stringValue)
			if stringValue == "" {
				return nil, false
			}
			items = append(items, stringValue)
		}
		return normalizeStrings(items), len(items) > 0
	default:
		return nil, false
	}
}

func claimStringList(value any) ([]string, bool) {
	switch typed := value.(type) {
	case string:
		return normalizeStrings(strings.FieldsFunc(typed, func(r rune) bool { return r == ' ' || r == ',' })), true
	case []any:
		items := make([]string, 0, len(typed))
		for _, entry := range typed {
			stringValue, ok := entry.(string)
			if !ok {
				return nil, false
			}
			items = append(items, stringValue)
		}
		return normalizeStrings(items), true
	default:
		return nil, false
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
