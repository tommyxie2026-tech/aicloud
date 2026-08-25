# aicloud Documentation

`aicloud` is a hybrid private AI cloud platform centered on governed hybrid
model access, policy-aware Agent workflows and controlled execution.

```text
AI Cloud
  = Enterprise AI Control Plane
  + replaceable Gateway Data Plane
  + Agent Execution Cloud
```

Design principle:

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

## Start here

1. [Product positioning](aicloud-positioning.md)
2. [AI Cloud architecture](architecture/AI-Cloud-Architecture.md)
3. [v0.1 engineering design](design/v0.1-engineering-design.md)
4. [Module implementation plan and status](roadmap/AI-Cloud-Module-Implementation-Plan-and-Status.md)
5. [Durable Task workflow implementation plan](roadmap/2026-08-25-durable-task-workflow-implementation-plan.md)

## Current implementation-stage documents

The next stage moves the existing synchronous governed model runtime to a
durable, restart-safe Task workflow.

| Document | Authority |
|---|---|
| [Durable Workflow Engineering Contract](design/durable-task-workflow-engineering-contract.md) | package boundaries, identities, transactions and error taxonomy |
| [Task State Machine](design/agent-state-machine.md) | persisted states and allowed transitions |
| [Database Schema](design/database-schema.md) | Task events, optimistic concurrency, Outbox and cost deduplication |
| [API Specification v1](api/api-spec-v1.md) | API-wide conventions and compatibility |
| [Task Workflow API v1](api/task-workflow-api-v1.md) | asynchronous Task, events and cancellation contract |
| [Agent Runtime Design](design/Agent-Runtime-Design.md) | API/Worker/workflow/activity execution boundaries |
| [Temporal Development and Test Plan](development/temporal-development-and-test-plan.md) | local profile, Worker, CI and failure injection |

## Architecture and design

- [Product architecture](aicloud-product-architecture.md)
- [Model Registry](design/Model-Registry-Design.md)
- [Tool Gateway](design/Tool-Gateway-Design.md)
- [Sandbox](design/Sandbox-Architecture.md)
- [Evaluation Platform](design/Evaluation-Platform-Design.md)
- [API and Data Model](design/api-and-data-model-design.md)
- [Kubernetes deployment](deployment/kubernetes-architecture.md)
- [Repository structure](development/repository-structure.md)

## Operational baseline

- [Persistence and routing](development/persistence-and-routing.md)
- [Trace, evaluation, fallback and admission](operations/trace-evaluation-fallback-admission.md)
- [Secure Tool execution](security/secure-tool-execution.md)

## Current engineering baseline

Implemented foundations include:

```text
Go modular monolith
API server and Worker entrypoints
memory and PostgreSQL repositories
governed Model Registry and Router
model runtime fallback and circuit breaker
trace, evaluation and cost evidence
Docker Compose and Helm baseline
```

Durable Task events, transactional workflow-start Outbox, Temporal execution,
cancellation and restart recovery are specified but remain the next code stage.
