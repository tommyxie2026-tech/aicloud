# GitOps Integration

## 1. Goal

The `integrations/gitops` package converts policy-evaluated infrastructure proposals into deterministic, reviewable manifest patch plans, dry-run manifest write results, and dry-run branch/commit/PR plans.

It is the bridge between:

```text
Evaluated ChangeProposal
```

and:

```text
GitOps-ready manifest change
```

It does not directly apply changes to live infrastructure.

## 2. Core Boundary

Allowed flow:

```text
ModelGateway
  ↓
ChangePlan
  ↓
SafetyGuard / SchemaValidator
  ↓
ChangeProposal
  ↓
PolicyChecker
  ↓
Evaluated ChangeProposal
  ↓
ManifestPatchPlan
  ↓
DryRunManifestWriter
  ↓
BranchPlan / CommitPlan / PullRequestPlan
  ↓
future Git branch / commit / PR integration
  ↓
GitOps controller
  ↓
Infrastructure controller reconcile
```

Forbidden flow:

```text
Model or agent
  ↓
kubectl apply / helm upgrade / terraform apply / machine power operation
```

## 3. Current Files

```text
integrations/gitops/patch_plan.go
integrations/gitops/managedcluster_patch.go
integrations/gitops/manifest_writer.go
integrations/gitops/branch_plan.go
integrations/gitops/patch_plan_test.go
integrations/gitops/managedcluster_patch_test.go
integrations/gitops/manifest_writer_test.go
integrations/gitops/branch_plan_test.go
```

## 4. ManifestPatchPlan

`ManifestPatchPlan` is an auditable intermediate representation.

It contains:

```text
RequestID
ProposalID
Target
SourcePath
OutputPath
Changes
Rollback
Validation
PolicyResult
PRMetadata
```

It is created from a policy-evaluated `ChangeProposal`.

## 5. PatchPlanner

`PatchPlanner` converts `proposal.ChangeProposal` into `ManifestPatchPlan`.

Current allowlist:

```text
spec.workers[name=gpu-workers].replicas
```

Blocked fields:

```text
status
metadata.finalizers
metadata.ownerReferences
spec.credentials
spec.secretRef
spec.bmcSecretRef
```

Important behavior:

```text
- rejects nil proposal
- rejects unevaluated proposal
- rejects missing source path
- rejects missing changes
- rejects blocked fields
- rejects fields outside the allowlist
- generates rollback as inverse patch
- generates dry-run PR metadata
```

## 6. PRMetadata

`PRMetadata` contains:

```text
BranchName
CommitMessage
Title
Draft
```

Example:

```text
BranchName:    aicloud/request-001/scaleout/dev-gpu-cluster
CommitMessage: aicloud: ScaleOut ManagedCluster/dev-gpu-cluster
Title:         ScaleOut ManagedCluster/dev-gpu-cluster
Draft:         false
```

`Draft` is derived from deterministic policy approval requirement.

## 7. ManagedCluster Object Patcher

`ApplyManagedClusterPatch` applies a `ManifestPatchPlan` to an in-memory `infraapi.ManagedCluster` object.

It does not:

```text
- read YAML files
- write YAML files
- create commits
- create PRs
- call Kubernetes
- call GitOps controllers
```

Current supported field:

```text
spec.workers[name=gpu-workers].replicas
```

Important behavior:

```text
- validates input ManagedCluster
- validates target kind/name/namespace
- validates current value equals patch `from`
- applies new value from patch `to`
- rejects unsupported fields
- rejects missing worker group
- rejects negative replicas
- returns a new updated object
```

## 8. DryRunManifestWriter

`DryRunManifestWriter` converts:

```text
ManifestPatchPlan + ManagedCluster
```

into:

```text
WriteResult
```

`WriteResult` contains:

```text
SourcePath
OutputPath
Summary
Updated ManagedCluster
Changes
Rollback
```

It does not:

```text
- read files
- write files
- create commits
- create PRs
- call Kubernetes
```

## 9. BranchPlan / CommitPlan / PullRequestPlan

`BuildBranchPlan` converts:

```text
ManifestPatchPlan + WriteResult
```

into:

```text
BranchPlan
```

`BranchPlan` contains:

```text
BaseBranch
HeadBranch
CommitPlan
PullRequestPlan
```

This is dry-run metadata only.

It does not:

```text
- create a Git branch
- create a Git commit
- push to remote
- create a GitHub PR
- merge a PR
```

## 10. Current First Scenario

Input intent:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

Current example manifest:

```text
examples/infra/managedcluster-dev-gpu.yaml
```

Patch field:

```text
spec.workers[name=gpu-workers].replicas
```

Forward patch:

```text
3 -> 6
```

Rollback patch:

```text
6 -> 3
```

## 11. Safety Rules

GitOps integration must fail closed if:

```text
- proposal is not policy evaluated
- policy result is missing
- field is outside allowlist
- field is blocked
- target manifest does not match proposal target
- current value does not match expected `from`
- generated object fails API validation
- base branch is missing
- head branch metadata is missing
- output path is missing
```

## 12. Current Tests

```text
integrations/gitops/patch_plan_test.go
integrations/gitops/managedcluster_patch_test.go
integrations/gitops/manifest_writer_test.go
integrations/gitops/branch_plan_test.go
```

Covered behavior:

```text
- evaluated proposal creates ManifestPatchPlan
- rollback patch is inverse of forward patch
- PR metadata is generated
- draft flag follows approval requirement
- unevaluated proposal is rejected
- blocked field is rejected
- unknown field is rejected
- source path is required
- ManagedCluster patch 3 -> 6 succeeds
- original ManagedCluster object is not mutated
- current value mismatch fails closed
- target mismatch fails closed
- unsupported field fails closed
- missing worker group fails closed
- negative replicas fails closed
- dry-run writer returns updated object and summary
- dry-run writer propagates patch errors
- branch plan includes base/head branch
- branch plan includes commit file change
- branch plan includes PR title/body/draft flag
```

## 13. Not Done Yet

```text
- YAML parser / writer
- stable YAML formatting
- multi-file manifest changes
- real branch creation
- real commit creation
- GitHub PR creation
- Argo CD / Flux integration
- live cluster apply
```

## 14. Recommended Next Steps

Recommended next steps:

```text
1. Run or verify go test ./... status.
2. Add provider config loading.
3. Add OpenAI-compatible HTTP client.
4. Add private/self-hosted provider config examples.
5. Add YAML read/write implementation only after dependency choice is clear.
6. Keep real GitHub PR creation separate from patch planning.
```
