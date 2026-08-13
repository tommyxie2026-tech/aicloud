# R7C API Authorization Boundary

## Status

Implementation slice: R7C RBAC + attribute-aware API authorization.

This layer runs after R7B authentication and before public API handlers. It does not replace PostgreSQL RLS, Task ownership checks, Tool Gateway policy, approval, or domain policy.

## Runtime flow

```text
HTTP Request
    |
    v
Request / Trace Metadata
    |
    v
OIDC/JWT or Trusted Ingress Authentication
    |
    v
Verified identity.Principal
    |
    v
API Authorization
    |
    +-- RBAC: may this role invoke this API action?
    |
    `-- ABAC: does the verified tenant/project context satisfy the resource scope?
    |
    v
Task ownership / PostgreSQL RLS / domain policy
    |
    v
Handler
```

Both RBAC and the attribute policy must allow the request. A downstream policy may further restrict an allowed request but may never widen an API authorization denial.

## Built-in roles

| Role | Intent |
|---|---|
| `tenant_admin` | Tenant administration and all currently mapped public API actions within the verified scope |
| `project_admin` | Project operations plus evaluation write |
| `developer` | Project read, Task create/route/model execution and Tool execution |
| `operator` | Operational project actions equivalent to the current v0.1 developer execution set |
| `reviewer` | Read access plus evaluation write |
| `viewer` | Read-only access |

No role means deny.

`system` principals are not accepted by the public authorization path. Internal SystemPrincipal operations require a dedicated trusted entry point and explicit capability/purpose contract.

## API actions

The v0.1 API boundary maps public routes to stable actions rather than authorizing by arbitrary URL strings. Current actions include model read/write, admission read/write, Task read/create/route/model execution, route/cost/audit/trace/evaluation reads, evaluation write, Tool read, and Tool execution.

The route map is explicit and fail closed:

- unknown `/api/` paths return `NOT_FOUND` before a business handler runs;
- unsupported methods on known resources return `METHOD_NOT_ALLOWED`;
- a missing authorization service returns `AUTHORIZATION_NOT_CONFIGURED`;
- evaluation errors return retryable `AUTHORIZATION_UNAVAILABLE`;
- denied decisions return `FORBIDDEN`.

Health and readiness endpoints remain outside the authenticated public API boundary.

## Scope policy

The v0.1 ABAC layer evaluates the verified Principal against the declared API resource scope:

- tenant scope requires `tenant_id`;
- project scope requires both `tenant_id` and `project_id`;
- when an explicit target tenant/project is supplied to the policy, it must match the verified Principal;
- unsupported resource scope is a configuration error.

For Task IDs, this API policy is not the source of ownership truth. `taskScopeGuard`, tenant-aware repositories and PostgreSQL FORCE RLS remain independent enforcement boundaries and resolve the actual Task ownership. This preserves defense in depth and prevents RBAC/ABAC from weakening storage isolation.

## Separation from domain policy

`internal/authorization` governs whether an authenticated caller may invoke a public API operation.

`internal/policy` continues to govern context-specific execution decisions such as Tool Gateway policy and approval. The two contracts are intentionally separate:

```text
Authentication -> API RBAC/ABAC -> Domain/Tool Policy -> Approval -> Execution
```

API authorization answers whether the caller may request an operation. Domain policy answers whether that operation is permitted under runtime business, risk and governance context.

## Role source

Roles and Principal attributes are consumed only from the verified `identity.Principal`. In OIDC mode they originate from cryptographically verified claims. The authorization layer does not parse JWTs or trust raw identity headers.

## Testing gate

R7C requires tests for:

- allowed role/action combinations;
- read-only role mutation denial;
- missing role denial;
- tenant/project attribute mismatch denial;
- public SystemPrincipal denial;
- route/action mapping for every currently implemented public endpoint;
- unknown path and method fail-closed behavior;
- authentication-before-authorization middleware ordering;
- existing Task scope/RLS tests remaining green.

R7D will converge request/response schemas and executable OpenAPI contracts. R7C does not change the frozen Task aggregate or R6 transaction semantics.
