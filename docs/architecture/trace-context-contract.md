# AI Cloud Trace and Correlation Context Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define stable identifiers and propagation rules so every Task can be reconstructed across API, workflow, model, policy, tool, sandbox, audit, cost and evaluation systems.

## 2. Canonical Correlation IDs

```text
request_id
trace_id
tenant_id
project_id
task_id
workflow_id
agent_id
model_attempt_id
tool_invocation_id
policy_decision_id
approval_id
evaluation_run_id
```

Not every operation has every ID, but IDs that exist must be propagated consistently.

## 3. Identity Hierarchy

```text
Request
  └─ Task
      └─ Workflow
          ├─ Agent step
          │   ├─ ModelAttempt
          │   └─ ToolInvocation
          ├─ PolicyDecision
          ├─ Approval
          └─ EvaluationRun
```

`task_id` is the primary business correlation key. `trace_id` is the primary telemetry correlation key.

## 4. Generation Rules

- `request_id` is generated or accepted from trusted gateway format at the API boundary.
- `trace_id` follows OpenTelemetry/W3C Trace Context.
- `task_id` is generated once at Task creation and never changes.
- attempt/invocation/decision IDs are unique immutable record identifiers.
- no module creates a replacement Task or Trace identity simply because a retry occurs.

## 5. Context Propagation

A normalized execution context carries:

```yaml
context:
  principal: ...
  tenant_id: string
  project_id: string
  request_id: string
  trace_id: string
  task_id: string?
  workflow_id: string?
```

HTTP uses W3C `traceparent`/`tracestate` for distributed tracing. Internal workflow/activity messages explicitly carry business correlation IDs even when telemetry context is reconstructed.

## 6. Logging

Structured logs for Task-related execution include at minimum:

```text
timestamp
level
component
request_id
trace_id
tenant_id
project_id
task_id
operation
message
```

Sensitive prompt/data/credentials are not logged by default.

## 7. OpenTelemetry Span Model

Recommended hierarchy:

```text
HTTP request
  -> task.command
    -> workflow.start/signal
      -> agent.step
        -> router.decide
        -> model.attempt
        -> policy.decide
        -> tool.invoke
          -> sandbox.execute
        -> validation
        -> evaluation
```

Span attributes include IDs and versions needed for investigation, but large payloads and secrets are stored as governed evidence references rather than span attributes.

## 8. Audit and Cost Linkage

AuditEvent and CostEvent must contain `tenant_id`, `project_id`, `task_id` when Task-related, plus `trace_id` or an equivalent evidence link.

This makes it possible to answer:

```text
What happened?
Who caused it?
Which model/tool was used?
Why was it allowed?
How much did it cost?
```

from one Task or Trace identity.

## 9. Retry/Fallback

Retries preserve Task and logical operation context but create new physical attempt IDs where appropriate.

```text
same task_id
same logical operation_id
new model_attempt_id
```

Fallback must remain inside the same Task/Trace lineage unless a new trace is intentionally linked using trace links.

## 10. Sampling

Security/audit/cost evidence is not subject to telemetry sampling. Trace sampling may reduce low-value spans, but required business evidence must still be persisted.

High-risk Tool operations and failures should be sampled at 100% for tracing unless prohibited by data policy.

## 11. Retention and Privacy

Identifiers are long-lived enough for audit correlation. Payload retention follows data policy separately. Trace/log systems must support redaction and tenant-aware access control.

## 12. Acceptance Criteria

- One Task ID reconstructs route, model attempts, policy, approval, tools, audit, cost and evaluation records.
- request/trace/task IDs survive API and Worker restart.
- retries do not create replacement Task identities.
- structured logs consistently include correlation context.
- OpenTelemetry context propagates across API/workflow/activity boundaries.
- sampled tracing never removes mandatory audit/cost/business evidence.
- secrets are absent from default traces/logs.