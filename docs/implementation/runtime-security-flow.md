# Runtime and Security Flow

## End-to-end task path

```text
1 Authenticate request
2 Resolve tenant/project/subject
3 Validate + idempotency check
4 Persist Task(CREATED) + TaskEvent
5 Start durable workflow
6 Load Agent/Workflow versions
7 Policy pre-check + budget pre-check
8 Router selects model version + fallback chain
9 Agent plans using model
10 Validate structured Plan
11 For each side-effecting step:
   a resolve Tool
   b schema validation
   c policy decision
   d approval if required
   e lease short-lived credential
   f execute Tool or Sandbox
   g filter output
   h append audit/cost/trace
12 Validate final result
13 Reconcile task cost
14 Persist terminal state
15 Run/schedule evaluation
```

## Routing contract
A RouteRequest includes task type, required capabilities, data classification, region/residency, maximum cost, latency/SLA target, inference effort preference, service tier preference, and tenant/project policy references.

Candidate eligibility is boolean and happens before scoring. A candidate that violates policy, license, residency, capability, admission state, health, or hard budget is never rescued by a high score.

Initial score is intentionally simple and explainable:

```text
score = quality_weight * quality_score
      + reliability_weight * reliability_score
      + latency_weight * normalized_latency
      + cost_weight * normalized_cost
```

Weights are policy/configuration, versioned with RouteDecision. Do not introduce ML-based routing before deterministic routing is measurable.

## Fallback
Fallback chain is produced at route time and revalidated before each attempt. Maximum attempts and total deadline are bounded. Fallback cannot cross a policy boundary. Retryable provider errors may trigger fallback; invalid request, policy denial, content rejection, or context/schema errors do not automatically retry another provider unless explicitly classified safe.

## Structured output
Plans and high-impact model outputs MUST use schema-constrained structured output when supported. Otherwise parse then validate against JSON Schema/domain validators. Invalid output never reaches Tool Gateway.

## Policy decision input

```text
subject identity + roles
 tenant/project
 task/agent/workflow version
 model/tool/action/resource
 data classification
 risk level
 requested side effect
 budget/cost estimate
 environment (dev/stage/prod)
 time/region
```

Policy output includes decision, reason code, constraints, policy version, and optional approval requirement.

## Approval
Approval records are immutable decisions. Workflow enters WAITING_APPROVAL and does not hold credentials while waiting. Approval has scope, expiry, approver identity, reason, and exact action/resource digest. If the action changes, approval is invalid and must be requested again.

## Credential lifecycle
Credentials are acquired only after Allow/approved decision, scoped to task/tool/resource/action, short-lived, delivered directly to the adapter/runtime, redacted from logs, and revoked/expired after use. Models never see raw credentials.

## Tool Gateway
Tools are registered with name, version, input/output schema, owner, risk level, allowed environments, credential type, side-effect classification, timeout, and audit requirements.

Side-effect classes: READ_ONLY, REVERSIBLE_WRITE, DESTRUCTIVE_WRITE, PRIVILEGED_ADMIN. Default policy for unknown class is deny.

## Sandbox
Generated code or shell-like operations execute in Sandbox, not API/worker host. Minimum Kubernetes profile: dedicated service account, runAsNonRoot, seccomp RuntimeDefault, read-only root filesystem, dropped capabilities, explicit CPU/memory/ephemeral storage, active deadline, network deny by default, explicit egress allowlist, task-scoped workspace, no hostPath, no Docker socket, no Kubernetes admin token.

## Prompt/tool injection boundary
Retrieved text and tool output are untrusted data. They cannot grant permission or override system/policy instructions. Tool selection is constrained by registered tool schemas and Policy Engine. Sensitive side effects require deterministic checks independent of model text.

## Trace hierarchy

```text
HTTP Request span
  Task span
    Workflow span
      AgentRun span
        RouteDecision span
        ModelCall span(s)
        ToolCall span(s)
          Policy span
          CredentialLease span
        Sandbox span(s)
        Validation span
        Evaluation span
```

Trace attributes include IDs and operational metadata, not secrets by default.

## Cost accounting
ModelCall, ToolCall, Sandbox, workflow runtime, retry/failure, storage/network, and optional human review emit CostEvents. Terminal Task performs reconciliation. Cost per successful task is the primary optimization metric.

## Failure semantics
All errors have retryable classification. Workflow retries only idempotent activities. Side-effecting Tool calls require an idempotency token or a reconciliation strategy. Unknown execution outcome transitions to a reconciliation step rather than blind retry.

## First reference scenario

```text
Goal: scale dev-gpu-cluster gpu-workers 3 -> 6
Model -> structured ChangePlan
Validator -> confirms target/from/to/risk/rollback
Policy -> environment=dev, reversible write, bounds check
Approval -> required only if configured threshold/risk requires it
Tool Gateway -> Kubernetes scale adapter
Credential Broker -> task-scoped credential
Controller -> patch expected resourceVersion
Validator -> read back replicas
Task -> COMPLETED
Audit/Cost/Trace -> reconciled
```

No model-generated command is executed directly.