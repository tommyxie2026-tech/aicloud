# AI Cloud Agent Runtime Design

## 1. Positioning

Agent Runtime is AI Cloud's controlled execution layer. It turns a durable Task
into model, policy, tool, sandbox and validation Activities without allowing
model output to directly access enterprise resources.

```text
Task Command
  -> Durable Workflow
      -> governed Model Runtime
      -> Policy + Tool Gateway
      -> Sandbox
      -> Validation + Evaluation
      -> Result, Trace and Cost
```

## 2. Deployment boundary

AI Cloud v0.1 remains a modular monolith:

- `cmd/api-server` accepts commands and serves resource views;
- `cmd/worker` hosts Temporal workflows and Activities;
- both compose shared internal packages;
- package separation does not imply separately deployed microservices.

## 3. Workflow separation

Three interfaces must remain distinct:

1. `WorkflowClient`: API/control-plane start, query and cancellation.
2. Temporal `TaskWorkflow`: deterministic orchestration only.
3. Activity implementations: database, model, tool, sandbox and telemetry I/O.

Temporal SDK types are confined to the adapter and Worker composition layers.
Domain state and repository interfaces remain workflow-engine neutral.

## 4. First workflow

```text
LoadTask
  -> PlanTask
  -> RouteModel
  -> InvokeModel
  -> optional ExecuteTool / RunSandbox
  -> ValidateResult
  -> RecordEvaluation
  -> ReconcileCost
  -> FinalizeTask
```

Each Activity receives a versioned DTO with Task, Workflow Run, trace, logical
Activity and attempt identities. Current policy is loaded by Activity rather
than trusted from old workflow history.

## 5. Durable state

PostgreSQL owns business Task state and append-only audit events. Temporal owns
execution scheduling, timers, retry and Signals. Every state transition uses
the transactional repository contract in
[Database Schema](database-schema.md).

Task creation uses a transactional outbox so a committed Task cannot be
silently orphaned when Temporal is unavailable.

## 6. Retry and cancellation

- model calls use bounded provider-aware retry and fallback;
- tools retry only when explicitly idempotent;
- sandbox infrastructure retry is separate from command retry;
- evaluation can retry using a stable source identity;
- cancellation is cooperative, bounded and followed by cleanup;
- replay and retry may not duplicate cost or external side effects.

## 7. Approval

High-risk actions transition to `WAITING_APPROVAL`. Approval arrives through a
Temporal Signal after API authentication and policy checks. The approval is
also persisted as an audit event. Expired, duplicate and stale Signals are
deterministically ignored.

## 8. Sandbox boundary

The core stage uses a fake Sandbox executor to prove policy, cancellation,
cleanup and artifact contracts. Kubernetes Job execution is an adapter and
stretch integration target. Ordinary unit tests never require a cluster.

## 9. Observability

```text
Task
  -> Workflow Run
      -> Activity
          -> Model / Policy / Tool / Sandbox / Evaluation / Cost
```

Every span/event carries:

- `task.id`;
- `workflow.id` and `workflow.run.id`;
- `trace.id`;
- `activity.id` and `activity.name`;
- `attempt.number`;
- model/tool/sandbox identifiers where applicable.

## 10. Security principles

- Agents and model output are untrusted.
- No Activity bypasses Policy or Tool Gateway for enterprise access.
- Activities receive task-scoped credentials, not long-lived user secrets.
- Workflow payloads and errors exclude credentials and sensitive raw provider
  data.
- Actor identity is derived from trusted authentication context.

## 11. Implementation and acceptance

The authoritative implementation sequence and gates are defined in:

- [Durable Workflow Engineering Contract](durable-task-workflow-engineering-contract.md)
- [Task State Machine](agent-state-machine.md)
- [Temporal Development and Test Plan](../development/temporal-development-and-test-plan.md)
- [Durable Workflow Implementation Plan](../roadmap/2026-08-25-durable-task-workflow-implementation-plan.md)
