# AI Cloud S0 Contract Freeze

## Purpose

Before additional implementation, freeze the contracts that prevent architectural drift.

## Core Principles

1. Task is the business execution aggregate.
2. Tenant and Project are security boundaries.
3. Workflow orchestrates execution but does not own business truth.
4. Provider is an adapter, not the model catalog.
5. Router selects among policy-approved candidates.
6. Policy decides permission; Router decides optimization.
7. Tool Gateway is the only side-effect boundary.
8. Every side effect requires evidence.
9. Every cost belongs to a Task.
10. Every production model requires admission and evaluation evidence.

## Runtime Ownership

PostgreSQL:
- business state;
- query state;
- resource ownership.

Workflow Engine:
- execution orchestration;
- retry and recovery.

Task Events:
- immutable business history.

Observability:
- operational reconstruction.

## Non-Goals

v0.1 does not attempt:

- microservice decomposition;
- autonomous unrestricted agents;
- provider-specific business logic;
- direct Agent access to enterprise resources.

## Review Gate

No Slice may start implementation unless:

- Domain contract reviewed;
- API contract reviewed;
- security boundary reviewed;
- persistence contract reviewed;
- test acceptance defined;
- English and Chinese documentation synchronized.
