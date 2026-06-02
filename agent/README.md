# Agent Layer

## 1. Goal

The `agent/` directory turns validated model output into controlled workflow artifacts.

It does not directly execute infrastructure changes.

The current agent-layer boundary is:

```text
Model output
  ↓
Validated ChangePlan
  ↓
ChangeProposal
  ↓
Deterministic PolicyChecker
  ↓
Evaluated ChangeProposal
  ↓
PR Draft
```

## 2. Core Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

Agent code may orchestrate planning and draft generation, but it must not bypass policy, approval, GitOps, or controller reconciliation.

## 3. Current Packages

```text
agent/proposal
agent/workflow
agent/prdraft
agent/pipeline
```

## 4. proposal

Path:

```text
agent/proposal/proposal.go
agent/proposal/proposal_test.go
```

Purpose:

```text
Convert a validated schema.ChangePlan into a workflow-ready ChangeProposal.
```

Important types:

```text
ChangeProposal
ProposalChange
PolicyResult
RollbackProposal
ValidationPlan
```

Important behavior:

```text
- ModelRiskHint is preserved only as a hint.
- PolicyResult is not copied from model output.
- ApprovalRequired is set only after deterministic policy evaluation.
```

## 5. workflow

Path:

```text
agent/workflow/workflow.go
agent/workflow/workflow_test.go
```

Purpose:

```text
Build a policy-evaluated ChangeProposal from a validated ChangePlan.
```

Flow:

```text
schema.ChangePlan
  ↓
proposal.FromChangePlan
  ↓
PolicyChecker.Evaluate
  ↓
ChangeProposal.ApplyPolicyResult
  ↓
Evaluated ChangeProposal
```

## 6. prdraft

Path:

```text
agent/prdraft/prdraft.go
agent/prdraft/prdraft_test.go
```

Purpose:

```text
Generate a reviewable PR title and body from an evaluated ChangeProposal.
```

Generated sections:

```text
Intent
Target
Proposed Changes
Risk and Approval
Rollback Plan
Validation Checklist
Safety Boundary
```

Important behavior:

```text
- PR draft generation requires PolicyResult.
- Unevaluated proposals are rejected.
- Draft text explicitly states that no direct infrastructure execution is performed.
```

## 7. pipeline

Path:

```text
agent/pipeline/pipeline.go
agent/pipeline/pipeline_test.go
```

Purpose:

```text
Run the first end-to-end planning-to-draft pipeline.
```

Current flow:

```text
Gateway.GeneratePlan
  ↓
ChangePlan
  ↓
WorkflowPlanner.BuildEvaluatedProposal
  ↓
Evaluated ChangeProposal
  ↓
PRDraftGenerator.Generate
  ↓
PR Draft
```

## 8. Current End-to-end Flow

The current tested chain is:

```text
MockProvider
  ↓
ModelGateway.GeneratePlan
  ↓
SafetyGuard
  ↓
BasicValidator
  ↓
ChangePlan
  ↓
WorkflowPlanner
  ↓
PolicyChecker
  ↓
Evaluated ChangeProposal
  ↓
PRDraftGenerator
  ↓
PR Draft
```

## 9. Tests

Current package tests:

```text
agent/proposal/proposal_test.go
agent/workflow/workflow_test.go
agent/prdraft/prdraft_test.go
agent/pipeline/pipeline_test.go
```

Run locally:

```bash
go test ./...
```

## 10. Not Done Yet

```text
- real GitHub PR creation
- GitOps manifest patch generation
- workflow state persistence
- approval workflow integration
- audit persistence
- retry / compensation workflow
```

## 11. Next Engineering Steps

Recommended next steps:

```text
1. Add integrations/github PR draft submitter interface.
2. Add manifest patch generation from ChangeProposal.
3. Add workflow state object.
4. Add approval state transition model.
5. Add persistent audit/event model.
```
