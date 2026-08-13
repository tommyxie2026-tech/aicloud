# R7B OIDC/JWT Verifier

## Status

Implementation slice: R7B authentication verification and server wiring.

This document extends the R7A `PrincipalVerifier` boundary with a production-oriented OpenID Connect / JWT verifier and explicit runtime verifier selection. RBAC/ABAC remains a separate R7C authorization boundary.

## Goal

Convert an external bearer token into a verified `identity.Principal` without allowing business handlers to parse or trust unverified claims.

```text
HTTP Request
    |
    v
Request / Trace Metadata
    |
    v
PrincipalVerifier
    |
    +-- trusted_ingress  (explicit compatibility mode)
    |
    `-- oidc
          |
          +-- HTTPS issuer discovery (optional)
          +-- HTTPS JWKS retrieval and cache
          +-- signing algorithm allow-list
          +-- RSA signature verification
          +-- issuer / audience validation
          +-- exp / nbf / iat validation
          +-- verified claim mapping
          v
    identity.Principal
          |
          v
Tenant / Project scoped API
```

## Security invariants

1. Issuer and audience are explicit configuration; neither is learned from the incoming token.
2. Discovery and JWKS endpoints must be absolute HTTPS URLs.
3. The discovery document issuer must exactly match the configured issuer.
4. `alg=none` and algorithms outside the configured allow-list are rejected.
5. R7B v0.1 supports `RS256`, `RS384`, and `RS512`; the default allow-list is `RS256` only.
6. RSA signing keys smaller than 2048 bits are ignored.
7. `kid` is required. Unknown keys trigger one JWKS refresh to support normal rotation, then fail closed.
8. `exp` is required. `nbf` and `iat` are validated when present with bounded clock skew.
9. Audience matching is exact and supports both scalar and array `aud` claims.
10. External JWTs may represent `user` or `service_account` principals only. `system` is always rejected.
11. Tenant/project/role/capability values are consumed only after signature and registered-claim validation succeeds.
12. Tokens are read only from one `Authorization: Bearer ...` header. Query-string or cookie token fallback is intentionally unsupported.
13. Bearer tokens are size bounded and must never be logged.
14. Authentication mode is explicit. An unknown mode prevents API-server startup.
15. In OIDC mode, incomplete or unreachable identity configuration prevents API-server startup instead of silently falling back to trusted headers.
16. Request/trace metadata wraps authentication so authentication failures can use the same correlation contract as authenticated requests.

## Runtime configuration

`AICLOUD_AUTH_MODE` selects the verifier:

- `trusted_ingress`: explicit v0.1 compatibility mode;
- `oidc`: cryptographic bearer-token verification.

OIDC configuration:

| Environment variable | Default |
|---|---|
| `AICLOUD_OIDC_ISSUER` | none |
| `AICLOUD_OIDC_AUDIENCE` | none |
| `AICLOUD_OIDC_JWKS_URL` | discovery when omitted |
| `AICLOUD_OIDC_ALLOWED_ALGORITHMS` | `RS256` |
| `AICLOUD_OIDC_TENANT_CLAIM` | `tenant_id` |
| `AICLOUD_OIDC_PROJECT_CLAIM` | `project_id` |
| `AICLOUD_OIDC_ROLES_CLAIM` | `roles` |
| `AICLOUD_OIDC_CAPABILITIES_CLAIM` | `capabilities` |
| `AICLOUD_OIDC_PRINCIPAL_TYPE_CLAIM` | `principal_type` |
| `AICLOUD_OIDC_SESSION_CLAIM` | `sid` |
| `AICLOUD_OIDC_CLOCK_SKEW_SECONDS` | `60` |
| `AICLOUD_OIDC_JWKS_CACHE_TTL_SECONDS` | `300` |

The compatibility default is currently `trusted_ingress` so existing v0.1 deployments do not change authentication mode implicitly. Production environments should set the mode explicitly and use `oidc` unless a separately authenticated ingress is deliberately providing identity.

## Default claim mapping

| Platform field | JWT claim |
|---|---|
| `SubjectID` | `sub` |
| `TenantID` | `tenant_id` |
| `ProjectID` | `project_id` |
| `Roles` | `roles` |
| `Capabilities` | `capabilities` |
| `Principal.Type` | `principal_type` |
| `SessionID` | `sid` |
| `Issuer` | verified `iss` |
| `AuthnMethod` | constant `oidc_jwt` |

The claim names are configurable so enterprise identity providers can map existing schemas without coupling domain code to one vendor.

## JWKS lifecycle

The verifier fetches JWKS at construction time so startup/configuration validation can fail early. Keys are cached for a bounded TTL. A token signed by an unknown `kid` or a key that has just rotated causes one forced refresh and one retry of signature verification.

This provides safe normal key rotation without accepting arbitrary keys from the token itself.

## Server wiring

The running API server constructs one verifier during startup and injects it into the R7A boundary:

```text
config.Load()
    |
    v
buildPrincipalVerifier()
    |
    v
WithRequestMetadata(
    WithPrincipalVerifier(
        verifier,
        FullHandler,
    ),
)
```

The startup path therefore fails closed on unsupported authentication modes or invalid OIDC initialization.

## Failure model

Authentication failures are returned to the R7A API boundary as verifier errors and become the stable `UNAUTHENTICATED` error envelope. OIDC metadata/JWKS configuration failures are startup/configuration failures, not per-request authorization decisions.

## Tests

The R7B core tests use an in-process TLS OIDC/JWKS server and real 2048-bit RSA signatures. They cover:

- successful verified claim mapping;
- exact audience rejection;
- expired-token rejection;
- external System-principal rejection;
- JWKS key rotation refresh;
- missing and duplicate Authorization-header rejection.

Configuration and server-selection tests additionally cover:

- explicit trusted-ingress compatibility mode;
- OIDC environment mapping and defaults;
- unsupported authentication mode rejection;
- incomplete OIDC configuration fail-closed behavior.

## Completion gate

R7B code and wiring are complete when full repository CI passes and the authentication-boundary review confirms the invariants above. Until that verification completes, the R7B PR remains Draft.

RBAC/ABAC remains R7C and OpenAPI/domain convergence remains R7D.
