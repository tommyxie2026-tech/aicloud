# AI Cloud CRD and API Design

## Objective

Define AI workloads as cloud native resources.

Inspired by Kubernetes declarative architecture.

## Core Resources

### Model CRD

```yaml
kind: Model
metadata:
  name: kimi-k3
spec:
  capability:
    coding: true
    reasoning: true
  provider: external
  costProfile: standard
```

### Agent CRD

```yaml
kind: Agent
metadata:
  name: code-agent
spec:
  model: kimi-k3
  tools:
    - github
    - shell
  sandbox: S2
```

### Workflow CRD

```yaml
kind: Workflow
metadata:
  name: software-change
spec:
  steps:
    - analyze
    - modify
    - validate
```

### Policy CRD

```yaml
kind: Policy
spec:
  requireApproval: true
  network: deny
```

## Design Principle

> Kubernetes manages compute resources. AI Cloud manages intelligent workloads.
