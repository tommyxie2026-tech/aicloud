# R1：Task Execution Contract 设计

[English](task-execution-contract.md) | **简体中文**

> 状态：R1 设计草案  
> 日期：2026-09-16  
> 上游研究：[AI Execution Control Plane](ai-execution-control-plane.zh-CN.md)

## 1. R1 目标

R1 不先构建复杂 Agent 平台，而是建立 AI Cloud 的统一执行契约，使现有推理、Agent、工具调用和未来 Human Task 都能够被描述成 Task，并获得统一的计划、运行、观测和审计语义。

R1 只定义三个核心对象：

```text
TaskDefinition → ExecutionPlan → ExecutionRun
```

- `TaskDefinition`：用户/系统想完成什么；
- `ExecutionPlan`：平台准备怎样完成；
- `ExecutionRun`：实际上怎样执行、消耗了什么、结果如何。

## 2. 设计原则

1. Task 是一级资源，Model/Agent 是执行能力；
2. Definition 描述意图，不绑定具体 Provider；
3. Plan 可变且可重新规划，Definition 保持稳定；
4. Run 是事实记录，不允许模型篡改历史；
5. Policy 在 Plan 与 Run 之间拥有最终授权权；
6. 所有执行必须产生可关联的 cost、latency、resource、outcome telemetry；
7. 契约必须兼容同步推理、长任务、Agent、多步骤 DAG 和 Human Approval；
8. API 先于 CRD，CRD 作为 Kubernetes-native 映射而非唯一接口。

## 3. TaskDefinition

建议 API：

```http
POST /v1/tasks
GET  /v1/tasks/{task_id}
POST /v1/tasks/{task_id}:plan
POST /v1/tasks/{task_id}:run
POST /v1/tasks/{task_id}:cancel
```

最小 Schema：

```yaml
apiVersion: aicloud.io/v1alpha1
kind: TaskDefinition
metadata:
  id: task-01J...
  tenant: tenant-a
  project: ops
spec:
  goal: "分析 StarRocks CN 写入阻塞并给出证据化根因候选"
  taskType: analysis
  inputs:
    - type: text
      ref: inline://request
    - type: file
      ref: artifact://logs/fe-audit.log
  constraints:
    maxCostUsd: 5.0
    timeoutSeconds: 1800
    dataClass: internal
    regionPolicy: apac-only
    humanApproval: on-high-risk-action
  successCriteria:
    - type: output-schema-valid
    - type: evidence-cited
  preferences:
    quality: high
    latency: normal
    cost: balanced
```

TaskDefinition 不出现 `openai`、`claude`、`vllm` 等具体 Provider，除非调用方显式声明硬约束。

## 4. ExecutionPlan

Plan 由 Planner + Router 产生，是“准备执行的方案”，必须经过 Policy 决策。

```yaml
apiVersion: aicloud.io/v1alpha1
kind: ExecutionPlan
metadata:
  id: plan-01J...
  taskId: task-01J...
spec:
  revision: 3
  generatedBy: planner/default
  steps:
    - id: classify
      type: model
      capability: fast-reasoning
    - id: inspect-logs
      type: tool
      tool: file.search
      dependsOn: [classify]
    - id: analyze
      type: model
      capability: deep-reasoning
      dependsOn: [inspect-logs]
    - id: validate
      type: evaluator
      profile: evidence-quality
      dependsOn: [analyze]
  routing:
    strategy: balanced
  estimated:
    costUsd: 1.40
    completionSeconds: 240
  policyDecision:
    status: allowed
    decisionId: pd-01J...
```

关键点：Step 优先引用 `capability`，而不是直接写死模型。Router 在执行前解析成具体 ModelProfile/AgentProfile/ResourceProfile。

## 5. ExecutionRun

Run 是不可变事实流的聚合视图：

```yaml
apiVersion: aicloud.io/v1alpha1
kind: ExecutionRun
metadata:
  id: run-01J...
  taskId: task-01J...
  planId: plan-01J...
status:
  phase: succeeded
  startedAt: 2026-09-16T14:50:00+08:00
  finishedAt: 2026-09-16T14:53:42+08:00
  steps:
    - id: analyze
      phase: succeeded
      resolvedExecution:
        provider: provider-x
        modelProfile: reasoning-large-v7
        runtime: remote-api
      usage:
        inputTokens: 22000
        outputTokens: 4100
        cacheReadTokens: 17000
        costUsd: 0.82
        latencyMs: 38200
  outcome:
    artifactRef: artifact://runs/run-01J/result.md
    success: true
  totals:
    costUsd: 1.17
    durationMs: 222000
    humanInterventions: 0
```

历史事件采用 append-only event log；`ExecutionRun.status` 是投影视图。这样后续可以 replay、audit、debug 和训练 procedural memory。

## 6. 状态机

```text
Task:
DRAFT → ACCEPTED → PLANNED → RUNNING → SUCCEEDED
                         │          ├→ FAILED
                         │          ├→ CANCELLED
                         │          └→ SUSPENDED → RUNNING
                         └→ REJECTED_BY_POLICY
```

Plan revision 与 Run attempt 分离：重新规划产生 `plan.revision+1`；同一 Plan 的基础设施重试产生新的 `attempt`，避免把“策略变化”和“运行失败”混在一起。

## 7. Event Model

R1 至少定义：

```text
task.created
plan.generated
policy.allowed
policy.denied
run.started
step.started
step.completed
step.failed
run.suspended
human.approval.requested
human.approval.received
run.completed
run.failed
run.cancelled
```

所有事件包含 `task_id / plan_id / run_id / trace_id / tenant / timestamp / actor`。

## 8. Telemetry Contract

统一 OpenTelemetry trace 语义，并增加 AI Cloud 属性：

```text
ai.task.id
ai.plan.id
ai.run.id
ai.step.id
ai.model.profile
ai.provider
ai.tool.id
ai.policy.decision_id
ai.resource.profile
ai.cost.usd
ai.tokens.input
ai.tokens.output
ai.tokens.cache_read
ai.human.interventions
ai.outcome.success
```

R1 的价值之一就是让未来 Router 能从真实 Task 数据学习，而不是依赖静态 Benchmark。

## 9. Policy 插入点

```text
TaskDefinition
     ↓
Planner
     ↓
Candidate Plan
     ↓
Policy Check ①：数据/模型/地域/预算
     ↓
Resolved Plan
     ↓
Policy Check ②：Tool/credential/action
     ↓
Runtime
     ↓
High-risk action ─→ Human Approval
```

R1 只需要定义 Policy Decision Contract，不要求一次实现完整策略语言。

## 10. Runtime Adapter

定义统一 SPI：

```text
ExecutionAdapter
├── ModelAPIAdapter
├── VLLMAdapter
├── SGLangAdapter
├── AgentAdapter
├── ToolAdapter
└── HumanAdapter
```

接口语义：

```text
prepare(step, context)
execute(step, context)
suspend(run)
resume(run)
cancel(run)
collect_usage(run)
```

这样后续 Codex-like harness、LangGraph、CrewAI 或自研 Agent 都是 Adapter，而不是平台核心依赖。

## 11. PoC：运维故障分析任务

R1 首个 PoC 建议选 AI Cloud 熟悉且可客观评价的“生产故障分析”任务：

```text
用户提交日志/现象
      ↓
TaskDefinition
      ↓
Planner
      ↓
Router 选择模型
      ↓
file/log tools
      ↓
deep reasoning
      ↓
evidence evaluator
      ↓
ExecutionRun
      ↓
结构化 RCA 报告
```

验收指标：同一 Task 可切换至少 2 个商业 Provider + 1 个本地模型；每次 Run 可追踪完整模型/工具/成本/延迟；失败可重试且不丢历史；预算超限可阻断；输出通过 evidence evaluator；所有结果可按 task_id replay。

## 12. R1 非目标

R1 暂不实现：复杂低代码 Workflow UI、自动长期记忆学习、完整 ABAC/Rego 策略系统、全量异构 GPU 调度、自动模型训练、跨组织 Agent Marketplace。

这些能力必须建立在稳定 Task Contract 之上。

## 13. R1 完成定义

R1 Done 必须同时满足：

```text
Task API 可用
+ Plan 可生成/版本化
+ Run 可执行/取消/重试
+ Provider-neutral Adapter
+ Policy Decision 插入点
+ OTel + Cost telemetry
+ Append-only execution events
+ 一个真实 PoC 端到端通过
```

完成后再进入 R2 Model Registry 2.0 + Intelligent Router 2.0。