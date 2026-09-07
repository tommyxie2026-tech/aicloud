# aicloud

`aicloud` is evolving toward an **AI Execution OS**: an enterprise execution control plane that turns goals into governed, verifiable, recoverable and learnable executions.

Core product direction:

```text
Goal
→ Execution
→ ExecutionPlan
→ ExecutionTarget
→ Evidence
→ Outcome
→ Experience
→ Improvement
```

The existing hybrid model gateway, provider abstraction, evaluation, policy and infrastructure-control capabilities remain important, but they are being repositioned as lower-level execution capabilities rather than the product center.

## Current R1 Kernel Work

The active R1 kernel now includes:

```text
Execution / Node state models
bounded DAG validation
Execution Conditions for parallel workflows
atomic budget reservation + settlement
Policy gating
scoped approvals with expiry/revocation
ExecutionTarget resolution
Supervisor runnable-node selection
Attempt lifecycle and Target invocation
Target snapshot/version lineage
Evidence / Outcome domain types
```

The first vertical slice will be an infrastructure/SRE incident investigation workflow using multiple execution targets, including private model capacity such as GLM-5.3 on PPU.

## Existing Repository Capabilities

The repository also contains earlier model-, agent-, policy-, infrastructure- and integration-oriented components. These are preserved and will be migrated incrementally behind the Execution contracts rather than rewritten wholesale.

## Engineering Principle

```text
Bounded before Autonomous
Deterministic before Learned
Verified before Trusted
Policy before Action
Evidence before Success
Experience before Self-Improvement
```

See `devdocs` and the companion `tommyxie2026-tech/devdocs` repository for architecture contracts and roadmap documents.
