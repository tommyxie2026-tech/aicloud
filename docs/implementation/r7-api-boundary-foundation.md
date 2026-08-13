# R7A API Boundary Foundation

## Status

Implementation slice for S2/R7-API. This document defines the first API/Auth convergence boundary after the R6 Task Transaction Kernel.

## Goal

Separate authentication mechanism from business handlers and establish stable request/trace correlation plus the canonical public error envelope before adding the production OIDC/JWT adapter and RBAC/ABAC policy binding.

## Runtime boundary

```text
HTTP Request
   |
   v
Request / Trace Metadata
   |
   v
Principal Verifier
   |
   v
Verified identity.Principal
   |
   v
Tenant / Project scoped handlers
```

Business handlers consume only `identity.Principal`. They must not parse JWT claims or raw identity headers directly.

## Principal verifier contract

`httpapi.PrincipalVerifier` is the API-boundary authentication abstraction. Implementations resolve one request into a validated `identity.Principal` or fail closed.

The existing trusted-ingress header mechanism is retained only as `TrustedIngressVerifier`, an explicit compatibility verifier. It does not represent the final production authentication model.

External verification must never create a System principal. System identity remains an internal explicit construction with purpose and capability requirements.

## Correlation contract

Public requests carry:

- `X-Request-ID`: request-level correlation identity;
- `X-Trace-ID`: end-to-end trace correlation identity.

Valid caller-provided values may be preserved. Missing or malformed values are replaced with generated values. The normalized values are written back to request and response headers and are available through request context.

## Error envelope

Authentication-boundary failures use the frozen OpenAPI shape:

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "...",
    "request_id": "req-...",
    "trace_id": "trace-...",
    "retryable": false
  }
}
```

R7A establishes the common envelope type and writer. Converting every legacy handler error path to this envelope remains part of R7-API convergence and is not claimed complete by this slice.

## Security decisions

1. Missing verifier fails closed.
2. Invalid external identity fails closed.
3. Health/readiness endpoints remain outside the public API authentication boundary.
4. Task APIs still require project scope.
5. Trusted ingress remains a compatibility mode only; production OIDC/JWT verification is the next slice.
6. Business handlers do not gain knowledge of authentication transport.

## Next slice

R7B adds the production OIDC/JWT verifier, issuer/audience/time/signature validation, verified claim-to-Principal mapping, and production wiring that removes raw identity headers as a trust source.
