# ADR-0010: Agent Runtime Plane Architecture

## Status

Accepted

## Context

AI Cloud is evolving from a model gateway into an enterprise AI operating platform.

The Control Plane manages model selection, governance, evaluation, and routing, but production AI workloads also require a controlled execution layer for agents, tools, memory, and workflows.

Therefore AI Cloud introduces a separate Agent Runtime Plane.

## Decision

AI Cloud adopts a two-plane architecture:

```text
                    AI Cloud Platform

        ┌────────────────────────────┐
        │      Control Plane          │
        │                            │
        │ Model Registry             │
        │ Evaluation                 │
        │ Governance                 │
        │ Policy Engine               │
        │ Intelligent Router          │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │      Execution Plane        │
        │                            │
        │ Agent Runtime              │
        │ Tool Gateway               │
        │ Memory Service             │
        │ Workflow Engine             │
        │ Sandbox Runtime             │
        │ Trace Collection            │
        └────────────────────────────┘
```

## Core Principles

### 1. Models do not execute actions directly

```text
Models propose.
Policy decides.
Controllers execute.
```

Agent output is untrusted until it passes:

- schema validation
- policy evaluation
- permission checks
- safety validation
- approval workflow when required

### 2. Tools are governed resources

Tools must be registered and controlled through Tool Gateway.

Example:

```yaml
tool:
  name: kubernetes-scale
  permission:
    scope:
      - dev-cluster
  approval:
    required: true
```

### 3. Agent identity is explicit

Every agent execution requires:

- identity
- owner
- delegated authority
- expiration time
- execution trace

### 4. Memory is a governed data service

Agent memory must support:

- data classification
- retention policy
- access control
- audit trail

## Agent Execution Flow

```text
User Request
      |
      v
Agent Planner
      |
      v
Policy Engine
      |
      v
Tool Gateway
      |
      v
Sandbox Execution
      |
      v
Validator
      |
      v
Controller
```

## Security Boundary

The runtime must enforce:

- network isolation
- short-lived credentials
- least privilege
- sandbox execution
- immutable audit logs
- external termination capability

## Multi-Agent Architecture

Future support:

```text
Planner Agent
      |
      +---- Coding Agent
      |
      +---- Security Agent
      |
      +---- Document Agent
      |
      +---- Verification Agent
```

Each agent may use different:

- models
- policies
- tools
- evaluation criteria

## Relationship with Control Plane

Control Plane decides:

- which model
- which provider
- which policy
- which security level
- which cost boundary

Runtime Plane executes:

- agent workflow
- tool invocation
- memory access
- sandbox operations
- trace collection

## Long Term Vision

AI Cloud becomes:

```text
AI Infrastructure Operating System

=

Model Control Plane
+
Agent Execution Plane
+
Governance Plane
```
