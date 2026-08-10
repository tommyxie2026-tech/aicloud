# AI Cloud Identity and Principal Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define the identity contract used by API, workers, repositories, policy, audit, tool execution and database access. The primary rule is:

> Absence of identity means unauthenticated. It never means system access.

## 2. Principal Model

```text
Principal
  ├─ User
  ├─ ServiceAccount
  └─ System
```

Every principal has an explicit type, stable subject ID and authenticated source.

```yaml
principal:
  type: user | service_account | system
  subject_id: string
  tenant_id: string?
  project_id: string?
  roles: []string
  capabilities: []string
  authn_method: oidc | workload_identity | internal_system
  issuer: string
  session_id: string?
```

A User or tenant-scoped ServiceAccount must have a tenant. Project-scoped operations also require a project. A System principal is explicit and is not inferred from missing tenant context.

## 3. Trust Establishment

```text
External Caller
  -> Authentication
  -> Verified Claims
  -> Principal Resolution
  -> Tenant/Project Resolution
  -> Authorization
```

Trusted ingress headers are allowed only as a v0.1 compatibility mechanism when the ingress authenticates the caller and overwrites the identity headers. Direct client-provided identity headers are never trusted in production.

## 4. Principal Types

### User

Human user authenticated through OIDC/OAuth2. The principal carries tenant membership and project assignment resolved from trusted identity data, not arbitrary request payload fields.

### ServiceAccount

Machine identity used by SDKs, CI/CD and platform integrations. Service accounts must use workload identity, short-lived JWTs or equivalent machine credentials and must have explicit scopes/capabilities.

### System

Internal platform identity used only by narrowly defined maintenance, migration, reconciliation or controller workflows.

System access requires all of:

- `principal.type = system`;
- a named system subject;
- a declared capability;
- an auditable reason/purpose;
- an execution path authorized for system principals.

## 5. Context Contract

Runtime context must carry a verified principal object. Domain code must not parse HTTP headers or JWT claims directly.

```go
type Principal struct {
    Type         PrincipalType
    SubjectID    string
    TenantID     string
    ProjectID    string
    Roles        []string
    Capabilities []string
    AuthnMethod  string
    Issuer       string
    SessionID    string
}
```

Expected helpers:

```text
WithPrincipal(ctx, principal)
PrincipalFromContext(ctx)
RequirePrincipal(ctx)
RequireTenant(ctx)
RequireProject(ctx)
RequireCapability(ctx, capability)
```

## 6. Authorization Layers

Authentication and authorization are separate concerns.

```text
Authentication
  -> Principal
  -> Resource Scope Check
  -> RBAC
  -> ABAC / Policy Engine
  -> Domain Operation
```

RBAC expresses coarse role permissions. Policy/ABAC evaluates contextual constraints such as data classification, model license, region, cost, tool risk and production environment.

## 7. System Access Rules

Forbidden patterns:

```text
ctx has no tenant -> allow system access
empty subject -> internal caller
header says system=true -> bypass authorization
```

Required pattern:

```text
Explicit System Principal
  + Explicit Capability
  + Explicit Authorized Entry Point
  + Audit Event
```

## 8. Persistence Boundary

Database access must not derive administrative privilege from an application-set boolean alone. Production roles should separate:

```text
aicloud_app_role       RLS enforced
aicloud_worker_role    RLS enforced
aicloud_admin_role     controlled maintenance
aicloud_migration_role schema migration only
```

Tenant operations use RLS-scoped transactions. Administrative access is performed by separate credentials/roles and produces audit evidence.

## 9. Audit Requirements

Every mutating operation records:

```text
principal_type
subject_id
tenant_id
project_id
roles/capability used
request_id
trace_id
resource
operation
decision
reason
```

System-principal activity must always be auditable.

## 10. Acceptance Criteria

- Missing identity always fails closed.
- A missing tenant never grants elevated access.
- User, ServiceAccount and System principals are distinguishable in code and audit.
- Project APIs fail without project scope.
- System access requires explicit capabilities.
- Cross-tenant access is rejected before domain mutation.
- Repository and DB tests verify both ordinary and administrative paths.

## 11. Migration from Current S1 Prototype

The current trusted-header scope contract is retained temporarily, but it must resolve into `Principal`; callers must not consume headers directly. Existing no-scope trusted system behavior is deprecated and must be removed before S2 is considered complete.