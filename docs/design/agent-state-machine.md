# Agent Task State Machine

> Applies to: AI Cloud v0.1 durable Task workflow
> State source of truth: PostgreSQL Task row plus append-only Task events
> Execution source of truth: Temporal workflow history

## 1. Persisted states

| State | Meaning | Terminal |
|---|---|---:|
| `CREATED` | Task request and workflow-start outbox are committed | No |
| `PLANNING` | Inputs, policy context and execution plan are being prepared | No |
| `EXECUTING` | Model, tool or sandbox activities are running | No |
| `WAITING_APPROVAL` | Workflow is waiting for an authorized signal | No |
| `VALIDATING` | Outputs, evidence and costs are being reconciled | No |
| `COMPLETED` | Validated result is available | Yes |
| `FAILED` | Current attempt ended unsuccessfully | Conditionally |
| `CANCELLED` | Cancellation was accepted and cleanup completed | Yes |

`RETRY` is an event, not a persisted state. Retrying a failed Task increments
the attempt and transitions the same Task from `FAILED` to `EXECUTING`.

## 2. Allowed transitions

| Current state | Allowed next states |
|---|---|
| `CREATED` | `PLANNING`, `CANCELLED`, `FAILED` |
| `PLANNING` | `EXECUTING`, `WAITING_APPROVAL`, `CANCELLED`, `FAILED` |
| `EXECUTING` | `VALIDATING`, `WAITING_APPROVAL`, `CANCELLED`, `FAILED` |
| `WAITING_APPROVAL` | `EXECUTING`, `CANCELLED`, `FAILED` |
| `VALIDATING` | `COMPLETED`, `EXECUTING`, `CANCELLED`, `FAILED` |
| `FAILED` | `EXECUTING` through an explicit retry event |
| `COMPLETED`, `CANCELLED` | none |

```text
CREATED -> PLANNING -> EXECUTING -> VALIDATING -> COMPLETED
   |          |            |             |
   |          |            +--> WAITING_APPROVAL --+
   |          |                                      |
   +----------+---------------------> CANCELLED <-----+
              |            |             |
              +------------+-----------> FAILED
                                           |
                                      retry event
                                           |
                                           v
                                       EXECUTING
```

## 3. Transition command

Every transition is a compare-and-swap operation against `resource_version`.
The repository must atomically append the Task event and update the
materialized Task row.

```go
type TransitionTaskCommand struct {
    TaskID              string
    ExpectedVersion     int64
    NextState           TaskStatus
    EventType           TaskEventType
    WorkflowRunID       string
    TraceID             string
    Attempt             int
    Actor               ActorRef
    Activity            string
    SourceEventID       string
    Error               *TaskError
    Metadata            map[string]string
}
```

The transition fails without writing when:

- the current state does not permit the requested next state;
- `ExpectedVersion` does not match;
- the event identity already belongs to a different operation;
- the Task is terminal;
- the actor or policy context is insufficient for an approval transition.

## 4. Cancellation

Cancellation is accepted from every non-terminal state.

1. API records a cancellation request and signals the stable Workflow ID.
2. Workflow requests cooperative cancellation of the active Activity.
3. Sandbox cancellation deletes the Job and collects available logs.
4. Cleanup Activities run with bounded timeouts.
5. Repository transitions the Task to `CANCELLED` exactly once.

Repeated cancellation returns the current Task and does not create another
state transition. `COMPLETED` and `CANCELLED` cannot be reopened.

## 5. Retry

Retry policy is Activity-specific. A retry records:

- a stable Activity ID and incremented attempt number;
- original error and retry decision;
- provider fallback or policy decision;
- incremental and cumulative cost;
- final outcome.

External side effects may retry only when the Tool or Sandbox contract declares
them idempotent or provides a durable deduplication key. Workflow replay alone
must not create a new cost or side effect.

## 6. Approval

Approval uses a Temporal Signal and a persisted audit event.

- only a trusted server/workload identity may be stored as an authenticated
  actor before OIDC is implemented;
- approval includes Task ID, expected resource version, policy decision ID,
  approver and expiration;
- stale or duplicate approval signals are ignored and audited;
- timeout follows policy and normally transitions to `FAILED` or `CANCELLED`.

## 7. Required tests

- every allowed transition succeeds;
- every omitted transition is rejected;
- concurrent transitions produce one winner and one version conflict;
- event append and Task update roll back together;
- retry increments attempt without changing Task or Workflow identity;
- cancellation is idempotent from every non-terminal state;
- terminal states reject further transitions;
- event replay reconstructs the materialized state and resource version.

## 8. Related documents

- [Durable Task Workflow Implementation Plan](../roadmap/2026-08-25-durable-task-workflow-implementation-plan.md)
- [Durable Task Workflow Engineering Contract](durable-task-workflow-engineering-contract.md)
- [Database Schema Design](database-schema.md)
- [Task Workflow API v1](../api/task-workflow-api-v1.md)
