# Task Execution Contract Research Draft (Superseded by ECP Convergence)

**English** | [简体中文](task-execution-contract.zh-CN.md)

> Status: **Superseded / historical research input**  
> Date: 2026-09-16  
> Current architecture: [Execution Control Plane Convergence](../docs/architecture/execution-control-plane-convergence.md)

## 1. Status

This document originally proposed a `TaskDefinition → ExecutionPlan → ExecutionRun` model. Code-level gap analysis later confirmed that AI Cloud already contains more mature canonical contracts protected by the S0 Frozen Contracts:

```text
internal/domain.Task
        ↓
execution.Goal (Task intent snapshot)
        ↓
execution.ExecutionPlan (versioned bounded DAG)
        ↓
execution.Execution
        ↓
execution.ExecutionAttempt
```

This document is therefore no longer an implementation authority and remains only as a record of research evolution.

## 2. Replaced concepts

| Early concept | Canonical contract | Decision |
|---|---|---|
| `TaskDefinition` | `internal/domain.Task` | Do not create a second Task aggregate |
| draft `ExecutionPlan` | `execution.ExecutionPlan` | Reuse the existing bounded-DAG contract |
| `ExecutionRun` | `execution.Execution` | Reuse the existing execution lifecycle |
| separate Execution Event Store | `domain.TaskEvent` | Reuse canonical business history |
| custom workflow engine | Temporal workflow | Reuse existing durable orchestration |
| `internal/taskexec` | `execution/` | Duplicate package removed |

## 3. Research conclusions retained

Task remains the first-class business resource; Providers, Models and Agents are execution capabilities; Policy owns final authorization; execution must retain durable history and cost/latency/resource/outcome telemetry; Provider, Agent Framework and Runtime should remain decoupled; and task economics should gradually move from `$ / token` toward Cost per Successful Task.

## 4. Current implementation path

Use `ECP-Cx` naming to avoid collision with the repository's existing R1–R7 remediation sequence:

```text
ECP-C1 Task ↔ Execution Binding
ECP-C2 Plan / Route convergence
ECP-C3 Persistent execution
ECP-C4 Policy / Approval / Budget convergence
ECP-C5 Evaluation / Outcome
ECP-C6 Organization Memory
```

The implementation authority is now `docs/architecture/execution-control-plane-convergence.md` together with the existing S0 Frozen Contracts.

## 5. Change control

Any future change to Task ownership, TaskEvent source of truth, Temporal ownership, or the canonical `execution/` domain must go through an architecture issue / ADR / migration review. It must not be introduced implicitly through a parallel package or schema.
