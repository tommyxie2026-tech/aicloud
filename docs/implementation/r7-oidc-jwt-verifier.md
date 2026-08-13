# R7B OIDC/JWT Verifier

## Status

Implementation slice: R7B authentication verifier core.

This document extends the R7A `PrincipalVerifier` boundary with a production-oriented OpenID Connect / JWT verifier. It does **not** yet declare the API server wiring or RBAC/ABAC layer complete.

## Goal

Convert an external bearer token into a verified `identity.Principal` without allowing business handlers to parse or trust unverified claims.

```text
Authorization: Bearer <JWT>
        |
        v
OIDCVerifier
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

## Failure model

Authentication failures are returned to the R7A API boundary as verifier errors and must become the stable `UNAUTHENTICATED` error envelope. Provider metadata/JWKS configuration failures are startup/configuration failures, not per-request authorization decisions.

## Tests

The R7B core tests use an in-process TLS OIDC/JWKS server and real 2048-bit RSA signatures. They cover:

- successful verified claim mapping;
- exact audience rejection;
- expired-token rejection;
- external System-principal rejection;
- JWKS key rotation refresh;
- missing and duplicate Authorization-header rejection.

## Remaining R7B work

Before R7B can be declared complete:

1. wire verifier selection into `cmd/api-server` configuration;
2. retain trusted-ingress mode only as an explicit compatibility mode;
3. ensure request/trace metadata wraps authentication so 401 responses are correlated;
4. add configuration and server-wiring tests;
5. run full CI and boundary review.

RBAC/ABAC remains R7C and OpenAPI/domain convergence remains R7D.
