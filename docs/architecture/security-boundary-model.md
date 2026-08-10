# AI Cloud Security Boundary Model

> Status: S0 Contract Freeze

## 1. Purpose

Define where trust is established, where authorization is enforced and where side effects are allowed. Security controls must be layered and independent from model behavior.

## 2. Trust Boundaries

```text
External Client
  -> Authn Boundary
  -> Principal / Tenant Boundary
  -> API Authorization Boundary
  -> Domain / Policy Boundary
  -> Workflow Boundary
  -> Tool Gateway Boundary
  -> Credential Boundary
  -> Sandbox / Adapter Boundary
  -> Enterprise Resource
```

No model, prompt or Agent output is trusted merely because it was produced by the platform.

## 3. Authorization Pipeline

```text
Authenticate
  -> Resolve Principal
  -> Resolve Resource Scope
  -> RBAC
  -> ABAC / Policy
  -> Domain Invariant
  -> Execute
```

Each layer may deny. Later layers cannot override a denial from a platform hard constraint.

## 4. Hard vs Contextual Policy

Hard platform constraints include:

- tenant isolation;
- explicit identity requirement;
- model license/admission restrictions;
- data residency requirements;
- prohibited tool/resource classes;
- sandbox baseline;
- credential TTL/max scope.

Contextual policy may add stricter tenant/project rules, approvals, budget thresholds or environment-specific requirements but cannot weaken hard constraints.

## 5. Agent Boundary

Agents do not receive long-lived production credentials and do not directly access enterprise systems.

Forbidden:

```text
Agent -> kubectl
Agent -> SSH key
Agent -> database password
Agent -> cloud admin token
Agent -> production filesystem
```

Required:

```text
Agent proposal
  -> Tool Gateway
  -> Schema validation
  -> Policy decision
  -> Approval when required
  -> Credential Broker
  -> Sandbox / Adapter
  -> Target Resource
```

## 6. Credential Boundary

Credential Broker issues task/tool-scoped, short-lived credentials after policy approval.

Every grant records:

```text
grant_id
tenant_id
project_id
task_id
tool_invocation_id
resource
permissions
expires_at
policy_decision_id
```

Credentials are delivered to the execution adapter/sandbox, not into LLM context.

## 7. Sandbox Baseline

Default production execution profile:

```text
non-root
read-only root filesystem
drop Linux capabilities
seccomp RuntimeDefault
resource limits
execution timeout
ephemeral workspace
default-deny network
no hostPath
no Docker socket
no cluster-admin token
```

Any exception requires an explicit higher-risk policy profile and approval evidence.

## 8. Data Boundary

Data is classified before model/tool access. Policy determines whether data may leave a tenant-controlled environment or be sent to a commercial provider.

```text
Data Classification
  + Tenant Policy
  + Provider/Deployment Residency
  + Model License/Admission
  -> Access Decision
```

Sensitive data must never be routed merely because a model has a higher quality score.

## 9. Model Boundary

Model output is untrusted input to downstream execution. Structured output must pass schema and semantic validation before any side effect.

For high-risk operations, model output produces a proposal/change plan, not an executable command.

## 10. Approval Boundary

Human approval binds to an immutable proposal digest, task, policy decision and expiration. If the proposal materially changes after approval, previous approval becomes invalid.

## 11. Audit Boundary

All security-relevant decisions are recorded, including denies. Required linkage:

```text
principal
resource scope
policy version
input/proposal digest
decision
reason
trace_id
task_id
```

## 12. Failure Behavior

Security infrastructure fails closed for mutating/high-risk operations. A policy engine, identity service or credential broker outage must not be converted into an allow decision.

Read-only degraded behavior, if supported, must be explicitly documented per endpoint.

## 13. Secrets

Secrets must not be:

- stored in prompt templates;
- persisted in TaskEvent payloads;
- logged in traces/audit raw payloads;
- returned in tool results unless explicitly redacted/authorized.

Secret references are preferred over secret values.

## 14. Acceptance Criteria

- Identity and tenant checks happen before protected resource access.
- Agent cannot directly obtain production credentials.
- Tool side effects require Tool Gateway and Policy.
- High-risk approved proposal is cryptographically/deterministically bound to approval evidence.
- Security subsystem failure fails closed for mutations.
- Sensitive data routing tests cover allowed and denied deployments.
- Sandbox baseline tests/policy checks prevent privileged execution.
- All denied and allowed high-risk actions produce audit evidence.