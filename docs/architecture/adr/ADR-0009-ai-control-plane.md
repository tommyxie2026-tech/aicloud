# ADR-0009: AI Cloud Control Plane

## Status

Accepted

## Context

The AI ecosystem now contains commercial APIs, open-weight models, private deployments, and specialized AI components.

Binding application architecture to a single provider creates vendor lock-in and limits future model evolution.

## Decision

AI Cloud adopts a Control Plane architecture.

Core components:

```text
Model Registry
Evaluation
Governance
Policy Engine
Intelligent Router
Provider Runtime
```

## Consequences

Positive:

- provider migration becomes possible
- models become replaceable assets
- routing can optimize quality, cost, and security
- governance becomes enforceable
- hybrid deployment is supported

Negative:

- architecture complexity increases
- evaluation infrastructure becomes mandatory
- policy management becomes a first-class engineering problem

## Future Direction

AI Cloud will evolve from:

```text
Model Gateway
```

into:

```text
Enterprise AI Operating System
```

with managed intelligence workloads, governance, and execution control.
