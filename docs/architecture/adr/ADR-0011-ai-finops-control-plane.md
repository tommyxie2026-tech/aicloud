# ADR-0011: AI FinOps Control Plane Architecture

## Status

Accepted

## Context

随着 AI Cloud 从模型管理平台演进为企业级 AI Operating System，Token 消耗、模型调用、Agent 执行链路和推理成本已经成为核心基础设施问题。

传统 Cloud FinOps 主要管理：

- CPU
- Memory
- Storage
- Network

AI Cloud FinOps 需要进一步管理：

- Token
- Model Cost
- Agent Execution Cost
- Tool Invocation Cost
- GPU Runtime Cost
- Business Value

因此 AI Cloud 需要引入独立 FinOps Control Plane。

---

# Architecture

```text
                    AI Cloud FinOps Plane

        ┌────────────────────────────┐
        │ Cost Intelligence           │
        │                            │
        │ Token Ledger               │
        │ Model Pricing              │
        │ Usage Analytics            │
        │ Budget Management          │
        │ ROI Analysis               │
        └────────────┬───────────────┘
                     │
                     ▼
              Policy Engine
                     │
                     ▼
             Intelligent Router
                     │
        ┌────────────┼────────────┐
        │            │            │
    Premium      Standard      Private
    Model        Model         Model
```

---

# Core Principles

## 1. AI Cost is Task Cost

不能只统计 Token。

真实成本：

```
Task Cost =

Input Token
+
Output Token
+
Reasoning Token
+
Tool Cost
+
Agent Loop Cost
+
GPU Runtime Cost
```

---

## 2. Cost-aware Routing

Router 不仅根据能力选择模型：

```
Capability
+
Quality
+
Security
+
SLA
+
Cost
```

综合决策。

---

# Cost Ledger

每次 AI 请求生成不可变成本记录：

```yaml
trace_id: xxx

model:
  name: coding-model

provider:
  name: openai

usage:
  input_tokens: 20000
  output_tokens: 5000

cost:
  token_cost: 0.15
  tool_cost: 0.02

business:
  application: code-review
```

---

# Budget Policy

支持：

- 企业预算
- 部门预算
- 应用预算
- Agent预算

示例：

```yaml
team:
  research:
    monthly_budget: 10000

application:
  chatbot:
    daily_budget: 100
```

---

# Model Economics

Registry 增加：

```yaml
pricing:
  input_token_price:
  output_token_price:

performance:
  avg_latency:
  success_rate:

cost_efficiency:
  success_per_dollar:
```

---

# Integration

AI Cloud Control Plane:

```text

Model Registry
       |
Evaluation
       |
Governance
       |
FinOps
       |
Policy Engine
       |
Intelligent Router
       |
Provider
```

---

# Long Term Vision

AI Cloud =

```
Model Control Plane
+
Agent Runtime Plane
+
Governance Plane
+
FinOps Plane
```

目标：

从管理模型调用

升级为

管理企业智能资源。
