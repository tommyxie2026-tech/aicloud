# Secure Tool Gateway and Sandbox Execution

## Purpose

AI Cloud treats every Agent-generated action as untrusted. Models and Agents cannot directly access enterprise systems, credentials, host filesystems, or Kubernetes APIs.

The mandatory execution path is:

```text
Task
-> Tool Registry
-> Policy Engine
-> Human Approval when required
-> Sandbox preflight
-> Task-scoped short-lived credential lease
-> Sandbox executor
-> Credential revocation
-> Immutable audit event
```

A denial, missing approval, invalid approval, unsafe Sandbox request, or incomplete dependency fails closed.

## Current implementation scope

The current executor is a `PlanningExecutor`. It generates a constrained Kubernetes Job and NetworkPolicy plan, but it does not submit resources or execute commands.

This makes the API path reviewable and testable before a Kubernetes controller is authorized to perform execution.

A later Kubernetes executor must consume the same validated `sandbox.Request` and preserve all controls described here.

## Tool Registry

Each Tool definition records:

- immutable Tool ID and version;
- container image;
- fixed command prefix;
- risk level;
- required permission;
- credential TTL;
- CPU and memory limits;
- execution timeout;
- network mode and allowed egress evidence;
- workspace write permission.

The initial built-in Tools are:

- `repo-inspect`: read-only repository inspection, no approval required;
- `workspace-command`: workspace write operation, explicit approval required.

These definitions are examples and do not constitute production-grade images or a complete Tool catalog.

## Policy

The Tool Gateway evaluates policy before issuing credentials.

The policy input includes:

- Agent identity;
- requested action;
- Tool resource.

The decision records:

- allowed or denied;
- whether human approval is required;
- reason;
- policy version;
- matched rule.

`DenyByDefault` is the fallback behavior. An OPA HTTP adapter is provided for external Rego decisions.

## Approval boundary

Approval records are bound to all of the following:

```text
Approval ID
+ Task ID
+ Tool ID
+ Action
+ Expiration
+ Revocation state
```

An approval cannot be reused for another Task, Tool, or action.

## Credential boundary

Credential leases are:

- bound to one Task;
- bound to one Tool;
- limited to one permission;
- short-lived;
- revoked after the Tool invocation returns;
- represented by an opaque lease reference rather than a raw secret.

The in-memory Broker is an interface-compatible development implementation. Production must replace it with Vault, cloud workload identity, Kubernetes projected credentials, or an equivalent broker.

## Sandbox controls

The generated Kubernetes plan enforces:

- dedicated Task namespace and ServiceAccount names;
- `automountServiceAccountToken: false`;
- non-root execution;
- RuntimeDefault seccomp;
- `allowPrivilegeEscalation: false`;
- `privileged: false`;
- all Linux capabilities dropped;
- read-only root filesystem;
- explicit CPU and memory requests and limits;
- active deadline and no retries;
- deterministic Job cleanup intent;
- ingress and egress denied by default;
- optional outbound access only through an approved egress-proxy namespace;
- workspace paths constrained under `/workspace`;
- raw secret-, token-, password-, or key-like environment variables rejected.

Network host allow-lists remain policy evidence. The production controller must route outbound traffic through an approved proxy instead of translating hostnames into broad CIDR access.

## HTTP API

List registered Tools:

```http
GET /api/v1/tools
```

Plan or execute a Tool for an existing Task:

```http
POST /api/v1/tasks/{taskId}/tools/{toolId}
```

Example read-only request:

```json
{
  "action": "inspect",
  "workspacePath": "/workspace/repo"
}
```

The API does not accept Task ID or Trace ID in the body. Both are loaded from the stored Task so callers cannot attach an invocation to a different execution trace.

Query Tool audit events:

```http
GET /api/v1/tasks/{taskId}/audit
```

Status behavior:

- `201`: request passed policy and produced a Sandbox result or plan;
- `400`: request or Sandbox constraints are invalid;
- `403`: policy denied the action;
- `404`: Task or Tool was not found;
- `409`: valid human approval is required.

## Audit evidence

Each invocation records:

- Task, Trace, Agent, Tool, and action;
- policy allow/deny result;
- policy version and matched rule;
- approval ID when used;
- credential lease ID;
- SHA-256 input digest;
- SHA-256 result digest;
- execution status and failure reason;
- timestamp.

Raw credentials are never stored in audit events.

## Security evaluation

Automated tests currently cover:

- default policy denial before credential issuance;
- approval required before high-risk actions;
- cross-Task approval rejection;
- cross-Task credential lease rejection;
- credential revocation after invocation;
- ServiceAccount token mounting disabled;
- privileged execution and privilege escalation disabled;
- default egress denial;
- raw secret-like environment rejection;
- workspace path escape rejection;
- HTTP audit evidence for planned and waiting-approval states.

## Remaining work

The following are required before production execution:

1. Kubernetes controller or worker implementing create, watch, collect, and destroy;
2. namespace, ServiceAccount, ResourceQuota, LimitRange, NetworkPolicy, and Job reconciliation;
3. signed workspace inputs and artifact outputs;
4. image signature, provenance, vulnerability, and malware admission;
5. Vault or workload-identity credential broker;
6. durable approval and audit persistence;
7. OPA policy bundle management and decision logging;
8. gVisor or Kata isolation profile for higher-risk Tools;
9. adversarial tests for prompt injection, dependency attacks, resource exhaustion, audit tampering, and Sandbox escape;
10. OpenTelemetry spans linking policy, approval, credential, Sandbox, and artifact activity.
