# aicloud Positioning

## 1. One-sentence Positioning

`aicloud` is a hybrid private AI cloud platform that connects public large models, private large models, and self-hosted open-source models through governed, auditable, policy-aware agent workflows.

## 2. Product Positioning

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

This means `aicloud` is not just a Kubernetes controller and not just a model API proxy.

It combines:

```text
- public model access
- private model access
- self-hosted open-source model access
- model gateway
- model routing
- model evaluation
- structured model output
- model safety boundary
- audit trail
- agent workflow
- infrastructure control scenarios
```

## 3. Target Users

Primary users:

```text
- enterprises that want AI capability but need private deployment
- platform engineering teams
- cloud-native infrastructure teams
- internal AI platform teams
- organizations that need controlled access to multiple models
```

Secondary users:

```text
- developers building internal agents
- SRE / DevOps teams
- security and compliance teams
- teams evaluating open-source models
```

## 4. Core Problems

Enterprises usually face these problems:

```text
1. Public models are powerful but may have data boundary concerns.
2. Private models are safer but fragmented and harder to evaluate.
3. Open-source models are flexible but require serving, routing, monitoring, and governance.
4. Different tasks need different models.
5. Agents need model access, but direct tool execution is risky.
6. Infrastructure changes require policy, audit, and approval.
```

`aicloud` solves this by providing a controlled hybrid model access layer plus a safe agent/control-plane boundary.

## 5. What aicloud Is

`aicloud` is:

```text
- a hybrid model gateway
- a private AI platform layer
- a provider abstraction layer
- a model routing and evaluation platform
- a safety and audit boundary for agents
- an AI-native infrastructure control plane foundation
```

## 6. What aicloud Is Not Initially

`aicloud` is not initially:

```text
- a foundation model training platform
- a GPU cluster scheduler only
- a simple API proxy only
- a chatbot only
- an uncontrolled autonomous agent
- a direct infrastructure executor
```

Training and fine-tuning may come later, but the first priority is safe hybrid model access and governed agent workflows.

## 7. Model Connectivity Positioning

### External model access

Used for:

```text
- strong reasoning
- complex planning
- coding assistance
- multilingual understanding
- architecture analysis
```

Provider type:

```text
Hosted / OpenAI-compatible public provider
```

### Internal private model access

Used for:

```text
- private data processing
- internal knowledge reasoning
- regulated workloads
- enterprise-only agent tasks
```

Provider type:

```text
Private / enterprise OpenAI-compatible endpoint
```

### Open-source model access

Used for:

```text
- low-cost inference
- local summarization
- private deployment
- specialized domain tasks
- offline or edge scenarios
```

Provider type:

```text
Local / self-hosted open-source provider
```

### Custom domain model access

Used for future narrow tasks:

```text
- infrastructure change risk classification
- Kubernetes YAML repair
- policy explanation
- validation report summarization
- runbook generation
```

## 8. Product Center

The product center is:

```text
Governed hybrid model access + policy-aware agent workflows
```

The infrastructure control plane is the first high-value application scenario, not the whole product boundary.
