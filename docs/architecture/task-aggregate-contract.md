# AI Cloud Task Aggregate Contract

> Status: S0 Contract Freeze

## 1. Purpose

Task is the core business execution unit and the aggregate root for governed AI work. All execution, routing, approval, tool invocation, cost, audit and evaluation evidence must be attributable to a Task.

## 2. Aggregate Shape

```text
Task
  ├─ Identity
  │   ├─ task_id
  │   ├─ tenant_id
  │   └─ project_id
  ├─ Request
  │   ├─ goal
  │   ├─ input
  │   ├─ agent_id/version
  │   └─ requested constraints
  ├─ Execution
  │   ├─ workflow_id
  │   ├─ status
  │   ├─ version
  │   └─ current step
  ├─ Result
  └─ Evidence references
```

Tenant and Project are immutable Task identity fields after creation. `created_by` records the Principal that created the Task.

## 3. Canonical Task Fields

```yaml
task:
  task_id: string
  tenant_id: string
  project_id: string
  created_by: string
  agent_id: string
  agent_version: string
  goal: string
  input: object
  constraints: object
  status: TaskStatus
  version: int64
  workflow_id: string?
  result: object?
  failure: object?
  created_at: timestamp
  updated_at: timestamp
  completed_at: timestamp?
```

Fields such as provider, model attempt, tool invocation and cost are not embedded as mutable Task state; they are separate Task-owned records/events.

## 4. State Machine

```text
CREATED
   ↓
PLANNING
   ↓
ROUTING
   ↓
EXECUTING
   ├───────────────┐
   ▼               │
WAITING_APPROVAL   │
   │               │
   └──────► EXECUTING
                    │
                    ▼
                VALIDATING
                    │
          ┌─────────┼─────────┐
          ▼         ▼         ▼
      COMPLETED   FAILED   CANCELLED
                         
Any non-terminal state may become FAILED, CANCELLED or EXPIRED when permitted by transition rules.
```

Terminal states:

```text
COMPLETED
FAILED
CANCELLED
EXPIRED
```

## 5. Transition Rules

Only the Task aggregate transition API may modify status.

Forbidden:

```go
task.Status = TaskCompleted
repo.Update(task)
```

Required semantic pattern:

```text
Task.Transition(command)
  -> validate current state
  -> validate actor/cause
  -> produce new Task state
  -> produce TaskEvent
  -> persist atomically
```

Each transition specifies:

- allowed source states;
- target state;
- command/cause;
- required principal/capability;
- required evidence;
- whether side effects are allowed before/after transition.

## 6. Versioning and Concurrency

Task uses optimistic concurrency with an integer `version`.

```text
UPDATE tasks
SET ..., version = version + 1
WHERE task_id = ? AND version = expected_version
```

A version conflict is retryable only after reloading current state and re-evaluating the command. Blind overwrites are forbidden.

## 7. Creation Invariants

Task creation requires:

- verified Principal;
- tenant and project scope;
- valid Agent reference or explicit system workflow type;
- goal/input satisfying API schema;
- idempotency key for public mutating API;
- generated request_id and trace_id.

Task creation and initial `TaskCreated` event must commit atomically.

## 8. Ownership

Long-term schema must contain `tenant_id`, `project_id` and `created_by` on `tasks`. The S1 `task_ownership` table is a migration bridge only and must not become the permanent source of Task identity.

Cross-project or cross-tenant Task moves are not ordinary updates. If supported later, they require an explicit migration operation and audit trail.

## 9. Failure Contract

Task failure is structured:

```yaml
failure:
  code: string
  category: validation | policy | provider | tool | sandbox | workflow | internal
  message: string
  retryable: bool
  source_ref: string?
  occurred_at: timestamp
```

Provider/tool attempt failures do not automatically mean the Task is failed; Workflow policy decides whether fallback/retry is possible.

## 10. Cancellation and Expiration

Cancellation is a command, not a field mutation. It must be idempotent and propagated to the durable workflow. Expiration is based on explicit deadline/TTL policy and emits an event.

External side effects that already completed must not be silently rolled back by Task cancellation; compensating workflows must be explicit.

## 11. Evidence Ownership

The following records must reference `task_id`:

```text
TaskEvent
RouteDecision
ModelAttempt
PolicyDecision
Approval
ToolInvocation
CredentialGrant
SandboxExecution
CostEvent
AuditEvent (when task-related)
EvaluationRun (production task evaluation)
```

## 12. Persistence Transaction

For business state changes:

```text
BEGIN
  validate expected task version
  update Task projection
  append TaskEvent
  append Outbox record when external delivery is required
COMMIT
```

Task state must never be committed without its corresponding TaskEvent.

## 13. Acceptance Criteria

- Direct arbitrary Task status mutation is impossible through public domain APIs.
- Invalid transitions fail deterministically.
- Every successful transition appends exactly one canonical state event.
- Task and initial event are atomically created.
- Optimistic concurrency rejects stale writes.
- Tenant/project identity cannot be changed by normal update APIs.
- Cancellation is idempotent.
- Terminal Task cannot re-enter a non-terminal state without a new explicit recovery/retry Task contract.

## 14. Implementation Impact

S2 must converge the Task API/schema to this contract. S3 implements durable workflow orchestration but must consume this domain state machine rather than owning business state directly.