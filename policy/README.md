# Policy Layer

## 1. Goal

The `policy/` directory contains deterministic decision logic for risk and approval.

Policy is the source of truth for:

```text
riskLevel
approvalRequired
matchedRule
policyResult
reason
```

Model output may provide a risk hint, but the model must not decide risk or approval by itself.

## 2. Core Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

## 3. Current Packages

```text
policy/checker
```

## 4. checker

Path:

```text
policy/checker/checker.go
policy/checker/checker_test.go
```

Purpose:

```text
Evaluate ChangeProposal using deterministic policy rules.
```

Important types:

```text
Checker
Policy
Rule
RiskLevel
PolicyError
```

Important functions:

```text
NewChecker
DefaultPolicy
Evaluate
Explain
ResultSummary
```

## 5. Current Default Policy

Current default rules:

```text
dev ManagedCluster small scale-out:
  environment = dev
  operationType = ScaleOut
  targetKind = ManagedCluster
  allowedField = spec.workers[name=gpu-workers].replicas
  maxReplicaDelta = 3
  riskLevel = Medium
  approvalRequired = false

staging ManagedCluster small scale-out:
  environment = staging
  operationType = ScaleOut
  targetKind = ManagedCluster
  allowedField = spec.workers[name=gpu-workers].replicas
  maxReplicaDelta = 3
  riskLevel = Medium
  approvalRequired = true
```

If no rule matches, policy fails closed:

```text
riskLevel = High
approvalRequired = true
matchedRule = fail-closed
result = REVIEW_REQUIRED
```

## 6. Current Policy Flow

```text
ChangeProposal
  ↓
Checker.Evaluate
  ↓
PolicyResult
  ↓
ChangeProposal.ApplyPolicyResult
```

## 7. Security Boundary

Policy must not rely only on:

```text
- model riskHint
- model explanation
- user-provided approval text
- PR title or body
```

Policy should rely on structured proposal fields:

```text
environment
operationType
target.kind
target.name
changed fields
replica delta
future: data sensitivity / RBAC / maintenance window
```

## 8. Tests

Current tests:

```text
policy/checker/checker_test.go
```

Test coverage:

```text
- dev small scale-out passes without approval
- staging small scale-out requires approval
- unknown field fails closed
- large replica delta fails closed
- nil proposal returns error
```

Run locally:

```bash
go test ./...
```

## 9. Not Done Yet

```text
- policy config loading
- policy versioning
- policy audit records
- RBAC / identity-aware policy
- environment-specific policy files
- maintenance window checks
- production approval chains
- OPA/Rego or CEL integration
```

## 10. Next Engineering Steps

Recommended next steps:

```text
1. Add policy file format.
2. Add policy loader.
3. Add policy version field to PolicyResult.
4. Add data sensitivity and environment guardrails.
5. Add production approval chain model.
```
