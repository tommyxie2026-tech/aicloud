# AI Cloud Workflow Source of Truth 契约

> 状态：S0 Contract Freeze

## 1. 目标

明确 PostgreSQL Domain State、TaskEvent Business History 与 Durable Workflow Runtime 三者的职责边界，使 Workflow 技术可以替换，而 Task Domain Model 不发生变化。

## 2. 三类 Source of Truth

```text
PostgreSQL
  负责当前 Business State / Query Projection

TaskEvent
  负责不可变 Business History

Temporal（或其他 Workflow Runtime）
  负责 Durable Execution History 与 Orchestration Progress
```

任何一层都不能悄悄替代其他层。

## 3. Canonical Ownership

### PostgreSQL

作为以下内容的 Canonical Store：

- 当前 Task Projection；
- Tenant/Project Ownership；
- Approval；
- Route Decision；
- Model Attempt；
- Tool Invocation；
- Cost/Audit/Evaluation Record；
- Idempotency Record；
- Outbox Record。

### TaskEvent

作为业务上有意义的事实与 Task Transition History 的 Canonical History。

### Workflow Runtime

负责执行机制：

- Activity Retry History；
- Timer；
- Durable Wait；
- Workflow Signal；
- Replay History；
- Activity Scheduling State。

Workflow History 不是 Business Database。

## 4. 控制方向

```text
API / Command
  -> Domain Transaction
     -> Task State + TaskEvent + Outbox
        -> Workflow Signal/Start
           -> Activities
              -> Domain Commands/Records
```

Activity 不允许直接任意改写 Task State，而必须调用遵循 Task Transition Contract 的 Application/Domain Service。

## 5. Workflow Identity

Task 可以保存 `workflow_id`，但 `task_id` 始终是业务 Aggregate Identity。

推荐确定性 Workflow ID：

```text
aicloud/task/{tenant_id}/{project_id}/{task_id}
```

映射必须唯一且可审计。

## 6. Determinism Rule

Workflow Code 必须 Deterministic。Network Call、Wall Clock、Random、Database Read 等非确定性行为必须在 Activity 或 Workflow-native Deterministic API 中完成。

会随时间变化的 Business Policy Result 必须持久化/版本化后传给 Workflow，而不能在 Replay 时悄悄重新计算。

## 7. Retry Ownership

Retry 分层：

```text
HTTP Command Retry      -> Idempotency Layer
Workflow Activity Retry -> Workflow Policy
Provider Fallback/Retry -> Model Runtime Policy
Tool Retry              -> Tool Execution Policy
```

某一层的 Retry 不得与另一层组合成指数级重复。每个 Operation 必须定义 Maximum Attempt Budget 与 Stable Idempotency Key。

## 8. Side Effect

Workflow Logic 不直接执行外部 Side Effect：

```text
Workflow
  -> Activity
    -> Tool Gateway / Provider Runtime
      -> Idempotent Side Effect
```

每一个 Side-effecting Activity 都必须有由 Task + Logical Operation + Attempt Policy 派生的稳定 Operation Key。

## 9. Restart 与 Recovery

系统必须能够承受：

- API Process Restart；
- Worker Restart；
- Workflow Worker Redeploy；
- PostgreSQL Temporary Outage；
- Provider/Tool Temporary Outage。

Recovery 不得重复执行外部动作。恢复后的 Task Projection 必须来自已提交 Domain State/Event，而不能依赖 Worker Memory。

## 10. Reconciliation

定期 Reconciler 对比：

```text
Task Projection
Workflow Runtime Status
TaskEvent History
```

检测到不一致时产生可审计 Reconciliation Finding，不允许静默修改历史。

## 11. Workflow 可替换性

Domain/Application Layer 只能依赖精简的 `workflow.Engine` Port，不得 Import Temporal SDK Type。Temporal-specific Type 只能存在于 Adapter Package。

建议 Port：

```text
StartTask
SignalTask
CancelTask
QueryExecutionState
```

该接口故意小于 Temporal API。

## 12. Completion Semantics

Workflow Completion 不等于 Task Completion。Workflow 可能异常结束但 Task 仍可恢复；Task 也可能先进入业务终态，后台 Evidence/Reconciliation 之后再完成。

Task Terminal State 只能由 Task Domain Transition Rule 决定。

## 13. 验收条件

- Domain Package 不依赖 Temporal SDK；
- 不读取 Workflow History 也可以查询 Task Business State；
- Workflow Replay 不会在缺少 Versioned Evidence 时重新计算可变 Policy/Evaluation Fact；
- API/Worker Restart 不产生 Duplicate Side Effect；
- 即使 Workflow Runtime Metadata 不可用，也能依赖 PostgreSQL + TaskEvent 做 Task Reconciliation；
- Activity Retry 使用 Stable Idempotency Identity；
- 替换 Workflow Runtime 不改变 Task/Policy/Router Contract。

## 14. 对实现的影响

S3 引入 Temporal Adapter 与 Worker。在进入 S3 之前，S2 必须先提供 TaskEvent、Outbox 与 Command Idempotency Primitive，避免 Workflow Coordination 建立在不安全的 Dual Write 上。