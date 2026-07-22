# Agent State Machine Design

## Lifecycle

```
CREATED
  |
PLANNING
  |
EXECUTING
  |
WAITING_APPROVAL
  |
VALIDATING
  |
COMPLETED
```

## Failure Handling

```
EXECUTING
   |
FAILED
   |
RETRY
   |
EXECUTING
```

## Principles

- Long running tasks must be resumable.
- Every transition must be observable.
- High risk operations require approval.
