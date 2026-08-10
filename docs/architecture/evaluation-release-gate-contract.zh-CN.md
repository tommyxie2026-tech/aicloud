# AI Cloud Evaluation 与 Release Gate 契约

> 状态：S0 Contract Freeze

## 1. 目标

定义如何把 Model、Prompt、Agent、Workflow 的质量证据变成可执行 Release Decision，而不是只停留在 Benchmark 报表。

## 2. Admission 与 Evaluation 分离

```text
Admission
  回答：这个 ModelVersion 在 License/Security 上是否有资格进入 AI Cloud？

Evaluation
  回答：这个 Version/Configuration 是否足够好，适合某个 Use/Release？
```

Admission 主要依据 License、Provenance、Artifact Integrity、Security Evidence；Evaluation 主要依据 Quality、Safety、Reliability、Latency、Cost 与 Task Outcome。

## 3. Evaluation Level

```text
L1 Offline Evaluation
  -> Golden Dataset / Regression

L2 Pre-Production Evaluation
  -> Shadow / Canary / Controlled Traffic

L3 Production Evaluation
  -> Real Task Trace / Outcome / Human Intervention / Cost
```

## 4. Evaluation Subject

Evaluation Identity 不能只写 Model Name，而应由一组 Versioned Configuration 组成：

```text
ModelVersion
PromptVersion
AgentVersion
WorkflowVersion
PolicyVersion
Tool/Sandbox Profile
DatasetVersion
EvaluatorVersion
```

必须保存真实执行时使用的完整组合，保证可复现。

## 5. EvaluationRun

```yaml
evaluation_run:
  evaluation_run_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  subject_ref: object
  dataset_id: string
  dataset_version: string
  evaluator_id: string
  evaluator_version: string
  config_digest: string
  metrics: object
  evidence_refs: []string
  started_at: timestamp
  completed_at: timestamp
```

## 6. Gate Metric

Gate 可以使用：

```text
quality score
safety score
reliability score
task success rate
hallucination/error rate
latency p50/p95
cost per successful task
human intervention rate
policy violation rate
tool failure rate
```

Threshold 必须 Versioned。

## 7. GateDecision

```yaml
gate_decision:
  gate_decision_id: string
  evaluation_run_id: string
  subject_ref: object
  decision: pass | fail | conditional
  threshold_version: string
  reasons: []string
  approved_for: []string
  expires_at: timestamp?
  created_at: timestamp
```

Mandatory Gate Fail 必须阻断 Promotion。Conditional Pass 必须明确限制条件，例如 Tenant、Traffic Percentage、Task Class、Expiration。

## 8. Promotion

Production Eligibility 必须同时满足：

```text
Admission Eligible
AND
Required Evaluation Gates Passed
```

Release/Promotion 本身属于受治理 State Transition，必须产生 Audit Evidence。

## 9. Regression

新 Version 需要与 Approved Baseline 对比。即使 Absolute Metric 达标，只要受保护 Metric Regression 超过 Tolerance，仍可判定 Fail。

## 10. Production Feedback

Production Task Trace 根据 Data Governance Rule 采样进入 Evaluation Candidate：

```text
Production Task
  -> Trace/Outcome
  -> Evaluation Dataset Candidate
  -> Regression Run
  -> GateDecision
  -> Router/Release Eligibility Update
```

形成生产反馈闭环。

## 11. Router Integration

Router 只能消费最终 Eligibility/Performance Evidence，不负责定义 Gate Threshold。

Mandatory Gate Fail 的 Candidate 在 Soft Routing Score 之前就必须被 Reject。

## 12. Data Governance

Evaluation Dataset 必须继承 Data Sensitivity 与 Tenant Rule。Production Data 不能自动进入 Global Benchmark Dataset，必须保留 Dataset Lineage、Consent、Retention Policy。

## 13. Reproducibility

每次 EvaluationRun 至少保存以下 Version/Digest：

- Subject Configuration；
- Dataset；
- Evaluator；
- Threshold；
- Relevant Runtime Parameter。

Release Decision 必须能从保存 Evidence 中复现。

## 14. Rollback

Production Monitoring 如果触发受保护 Metric Breach，可以执行 Rollback。Rollback 产生新的 Decision/Event，历史 GateDecision 不允许重写。

## 15. 验收条件

- Admission 与 Evaluation 使用不同 State/Record；
- Mandatory Gate Fail 阻止 Promotion/Route Eligibility；
- Evaluation Configuration 可通过 Version/Digest 复现；
- Baseline Regression Test 通过；
- Production Feedback 可以生成受治理 Evaluation Candidate；
- Tenant-sensitive Evaluation Data 未经 Policy 不得提升到 Global Scope；
- GateDecision 不可变且可审计；
- Rollback 可从 Evaluation Evidence 解释。