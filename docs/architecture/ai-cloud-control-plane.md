# AI Cloud Control Plane Architecture

## Overview

AI Cloud evolves from a model gateway into an AI infrastructure control plane.

The architecture principle is provider-independent, policy-driven, evaluation-driven model orchestration.

```text
                    AI Cloud Control Plane

        ┌──────────────┬──────────────┐
        │              │              │
 Model Registry   Evaluation     Governance
        │              │              │
        └─────── Policy Engine ───────┘
                       │
                       ▼
                Intelligent Router
                       │
          ┌────────────┼────────────┐
          │            │            │
       OpenAI       Gemini       Claude
          │
       Open Model
          │
       vLLM/SGLang
```

## Design Principles

### Provider Agnostic

Applications must not depend directly on a specific model vendor.

```text
Application
    ↓
AI Cloud API
    ↓
Provider abstraction
    ↓
Model execution
```

### Model Registry as Source of Truth

Registry manages:

- model identity
- capability
- version
- pricing
- license
- security classification
- lifecycle
- provider mapping

### Evaluation Driven

Evaluation is a continuous production feedback loop.

```text
Offline Evaluation
        ↓
Deployment
        ↓
Production Trace
        ↓
Quality Evaluation
        ↓
Routing Optimization
```

### Governance as Control Plane

Governance manages:

- security policy
- data policy
- license policy
- compliance requirements
- audit evidence

## Intelligent Router

Routing decisions consider:

```text
Capability
× Quality
× SLA
× Security
× Compliance
÷ Cost
```

The router selects the optimal model and provider according to workload requirements.

## Deployment Model

Supported execution targets:

```text
Commercial API
 - OpenAI
 - Gemini
 - Claude

Open Models
 - Qwen
 - DeepSeek
 - Llama
 - Mistral

Private Runtime
 - vLLM
 - SGLang
```

## Long Term Vision

AI Cloud becomes:

```text
AI Cloud = AI Infrastructure Control Plane

similar to:

Kubernetes -> compute workload management
AI Cloud   -> intelligence workload management
```
