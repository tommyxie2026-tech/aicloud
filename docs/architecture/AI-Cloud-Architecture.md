# AI Cloud Architecture Design

## 1. Vision

AI Cloud is a hybrid private AI cloud platform.

Positioning:

> Use mature Token Gateway components as the data plane, and build an enterprise AI Control Plane and Agent Execution Cloud above it.

The goal is not to build another model API proxy, but to provide enterprise-grade AI workload governance, execution and optimization.

## 2. Overall Architecture

```text
Users / Applications / IDE / Agents
                |
                v
        AI Cloud API Gateway
                |
                v
+--------------------------------+
| AI Control Plane                |
|--------------------------------|
| Tenant & Identity               |
| Policy Engine                   |
| Model Registry                  |
| Model Router                    |
| Evaluation Platform             |
| AI FinOps                       |
| Audit & Governance              |
+--------------------------------+
                |
                v
+--------------------------------+
| Token Gateway Data Plane        |
|--------------------------------|
| LiteLLM / Kong / Envoy          |
| Provider Adapter                |
| Token Accounting                |
| Rate Limit                      |
| Retry / Fallback                |
+--------------------------------+
                |
                v
+--------------------------------+
| Agent Execution Cloud           |
|--------------------------------|
| Workflow Engine                 |
| Agent Runtime                   |
| Tool Gateway / MCP              |
| Sandbox                         |
| Credential Vault                |
+--------------------------------+
                |
                v
Public Models / Private Models / Local Models
```

## 3. Core Principles

### Gateway is not the product

Token Gateway solves model connectivity. AI Cloud solves enterprise intelligence management.

### Task first, request first

The platform optimizes complete business tasks rather than individual model calls.

### Default deny

Agents are untrusted by default. Execution requires policy validation.

### Model independent

Support commercial APIs, private models and open-source models through unified protocols.

## 4. Core Planes

### Control Plane

Responsible for:

- identity
- policy
- model lifecycle
- evaluation
- cost governance
- audit

### Data Plane

Responsible for:

- model API compatibility
- routing
- token accounting
- resilience

### Execution Plane

Responsible for:

- long-running workflows
- Agent lifecycle
- tools
- sandbox execution

## 5. Relationship with Code Plan

Code Plan products are vertical AI applications built on top of AI Cloud.

```text
AI Cloud
 |
 +-- Developer AI Cloud
 |       |
 |       +-- Coding Agent
 |
 +-- Enterprise Agent Cloud
         |
         +-- Data Agent
         +-- Security Agent
         +-- Operation Agent
```

## 6. Safety Model

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```
