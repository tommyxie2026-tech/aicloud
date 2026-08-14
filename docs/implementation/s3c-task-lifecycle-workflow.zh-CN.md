# S3C Task 生命周期工作流契约

## 1. 目标

在保持 S3 已冻结归属模型的前提下，引入第一版确定性的 Task Lifecycle Workflow 与 Worker Registration Boundary：

```text
PostgreSQL  = 业务 / 查询状态
TaskEvent   = 追加式业务历史
Outbox      = 事务性投递意图
Temporal    = 持久编排 / 执行历史
```

S3C 需要证明 Workflow Definition 是确定性的、可重启恢复、可重放测试、显式注册，并与真实 Model / Tool / Infrastructure Side Effect 隔离。

本切片是 Workflow Kernel Proof，不是生产端到端 Task Execution 的正式激活。

## 2. 非目标

S3C 不做以下事情：

- 不在生产 Runtime 中把 API Process 的 Outbox Dispatcher 接到 Temporal；
- 不执行真实 Model Request；
- 不执行真实 Routing Decision；
- 不把 Policy 或 Approval 接成生产决策源；
- 不调用 Tool Gateway、Credential、Shell、Kubernetes、SSH、数据库变更工具或 Cloud API；
- 不建立由 Workflow Variable 所拥有的第二套 Task 状态机；
- 不让 Temporal 成为 Task Status 的事实源；
- 不把 Workflow State 暴露为新的公共 REST API；
- 不引入已废弃的 Build-ID / Legacy Worker Versioning API；
- 不宣称 Activity Exactly-Once Execution。

生产 PostgreSQL-backed Task Transition Activity，以及 Cancellation / Retry / Recovery 加固属于 S3D。

## 3. 领域模型变更

不增加或修改任何 Task Status。

权威 Task 状态仍然是：

```text
CREATED
PLANNING
ROUTING
EXECUTING
WAITING_APPROVAL
VALIDATING
COMPLETED
FAILED
CANCELLED
EXPIRED
```

Workflow 不创建另一套 Status Enum。它只消费 Activity 返回的 Task Status Snapshot，并通过 Activity Command 请求合法业务状态迁移。

### Workflow 身份

```text
workflow_type = task-execution-v1
workflow_id   = task/<task_id>
task_queue    = aicloud-task-v1
```

### Workflow 输入

稳定的 v1 输入只包含关联身份：

```go
type TaskWorkflowInput struct {
    SchemaVersion int
    TenantID      string
    ProjectID     string
    TaskID        string
    TraceID       string
}
```

规则：

1. `SchemaVersion` 从 `1` 开始且必填。
2. Tenant / Project / Task / Trace 全部必填。
3. Workflow Input 不携带 Credential、Provider Secret、无界 Model Payload 或可变 Authorization Claim。
4. Business State 必须通过 Activity 加载，不能复制进 Workflow Input 后长期作为事实源。

### Workflow 结果

Workflow 返回 Orchestration Evidence，而不是另一份业务 Projection：

```go
type TaskWorkflowResult struct {
    TaskID          string
    TraceID         string
    ObservedStatus  TaskStatus
    AlreadyTerminal bool
    Steps           []string
}
```

`ObservedStatus` 只是 Activity 最后观察到的状态，不覆盖 PostgreSQL 的权威地位。

## 4. API 变更

无。

公共 R7 API 与 OpenAPI Contract 保持不变。

S3C 不新增 Workflow Diagnostics、Signal、Query、Approval、Cancellation 或 Task Events 的公共 HTTP Route。

## 5. Worker 与注册契约

Worker Process 增加显式 Temporal 配置：

```text
AICLOUD_TEMPORAL_ENABLED=false
AICLOUD_TEMPORAL_ADDRESS=localhost:7233
AICLOUD_TEMPORAL_NAMESPACE=default
AICLOUD_TEMPORAL_TASK_QUEUE=aicloud-task-v1
AICLOUD_TEMPORAL_WORKER_STOP_TIMEOUT_SECONDS=30
```

S3C 中 `AICLOUD_TEMPORAL_ENABLED` 默认 `false`。

Disabled 时，不建立 Temporal Connection，也不启动 Polling Loop。

Enabled 时，如果 Address、Namespace、Task Queue、Registration 或 Temporal Client Initialization 非法，Worker 必须 Fail Fast。

### 稳定注册名

Workflow 与 Activity 使用显式字符串名，而不是依赖 Go Symbol Name：

```text
Workflow:
  task-execution-v1

Activities:
  aicloud.task.load.v1
  aicloud.task.transition.v1
  aicloud.task.plan.stub.v1
  aicloud.task.route.stub.v1
  aicloud.task.execute.stub.v1
  aicloud.task.validate.stub.v1
```

使用 Custom Name 时，Worker 必须关闭 Registration Aliasing，所有 Workflow / Activity 调用只使用稳定 External Name。

S3C Production Worker 的 Workflow Panic Policy 保持 SDK 的 Block 语义：Non-determinism 应阻塞等待修复，而不是自动把所有开放中的 Business Workflow 标记失败。

Worker Shutdown 必须 Graceful，并受配置的 Stop Timeout 限制。

## 6. Activity 边界

Workflow Code 与业务或外部状态的所有交互只能通过 Activity。

S3C 冻结两类 Activity。

### Business-state Seam

```go
type LoadTaskInput struct {
    TenantID  string
    ProjectID string
    TaskID    string
    TraceID   string
}

type TaskSnapshot struct {
    TaskID    string
    TraceID   string
    Status    TaskStatus
    Version   int64
    Terminal  bool
}

type TransitionTaskInput struct {
    TenantID        string
    ProjectID       string
    TaskID          string
    TraceID         string
    ExpectedVersion int64
    To              TaskStatus
    Cause           string
    OperationKey    string
}
```

Workflow 绝不能直接修改 Task，只能通过 Activity Seam 请求 Transition。

S3C Executable Test 中，这个 Seam 使用 In-Memory Fake，且必须执行与 Task Aggregate 相同的 Transition Rule 与 Idempotency Expectation。S3D 再替换为基于 PostgreSQL 的 Activity，接入既有 Aggregate / OCC / TaskEvent Transaction Kernel。

### Stub Execution Seam

S3C 中的 Planning、Routing、Execution、Validation Activity 从业务意义上都是确定性的 Test Double：不产生 Model、Tool、Network、Credential、Infrastructure 或外部 Database Side Effect。

它们只用于证明 Workflow Sequencing 与 Retry / Replay 行为。

S3C 中 Runtime API Process 仍不会自动把已提交 Task Dispatch 到这个 Worker，因此 Stub Lifecycle 不会意外成为生产 Task Executor。

## 7. 生命周期流程

参考编排：

```text
Workflow Start
  -> LoadTask
     -> 如果已经终态：返回 AlreadyTerminal
  -> Transition CREATED -> PLANNING
  -> PlanStub
  -> Reload/Transition PLANNING -> ROUTING
  -> RouteStub
  -> Reload/Transition ROUTING -> EXECUTING
  -> ExecuteStub
  -> Reload/Transition EXECUTING -> VALIDATING
  -> ValidateStub
  -> Reload/Transition VALIDATING -> COMPLETED
  -> Load final snapshot
  -> 返回 Orchestration Evidence
```

在 Transition Activity 返回已提交 / 已观察结果之前，Workflow 不得假设状态迁移成功。

每次 Mutation Request 使用最近一次 Task Snapshot / Transition Result 返回的 Version。

如果任一 Activity 观察到 Task 已终态，Workflow 必须 Short-Circuit，不能再请求状态迁移。

S3C Happy Path 不进入 `WAITING_APPROVAL`；只有在 S3E 接入 Policy / Approval Contract 后才引入 Approval Orchestration。

## 8. 确定性契约

Workflow Code 禁止直接使用：

- `time.Now` 或 Wall-clock Time；
- Random Generator 或 UUID Generation；
- Native Goroutine 进行 Workflow Control Flow；
- 直接 Network Call；
- Database Call；
- Provider / Tool / Credential Client；
- 会影响 Replay Decision 的 Environment Variable Read；
- Process-global Mutable State；
- 当迭代顺序影响 Workflow Command 时的 Map Iteration。

Workflow 只能使用 Temporal Workflow Primitive 与 Activity Result 做决策。

Workflow 使用稳定 External String Name 注册并调用 Activity。

### 向后兼容变更

当 S3C 生成第一份稳定 Workflow History Fixture 后，Checked-in Replay Test 必须成为 Merge Gate。

未来任何非向后兼容的 Workflow Definition Change，必须使用 `workflow.GetVersion` 等 Replay-safe Migration Mechanism，或者单独 Review 的 Worker Deployment Versioning Rollout。禁止引入已废弃的 Build-ID / Version-Set 机制。

## 9. Activity Retry 与 Timeout 契约

S3C 使用有界 Activity Option，不依赖无限重试。

Stub Lifecycle 初始默认值：

```text
StartToCloseTimeout = 10s
ScheduleToCloseTimeout = 60s
InitialRetryInterval = 1s
BackoffCoefficient = 2.0
MaximumRetryInterval = 10s
MaximumAttempts = 3
```

这些值属于 Execution Default，不是业务 SLA。

Activity Attempt 可以被重复执行，因此：

- Read Activity 必须无副作用；
- Transition Activity 必须接收稳定 `OperationKey` 与 ExpectedVersion；
- Stub Activity 必须可安全重复；
- Temporal Attempt Number 不能作为 Business Idempotency Identity。

S3D 在接入真实 PostgreSQL-backed Activity 后，可以按 Error Type 调整 Retry Class。

## 10. 幂等模型

S3C 继续保持此前已分离的身份：

```text
API command key
Outbox delivery key
Temporal Workflow ID
Business Activity OperationKey
```

Transition OperationKey 由 Task 与业务步骤确定性产生，例如：

```text
<task_id>:transition:PLANNING:lifecycle-v1
<task_id>:transition:ROUTING:lifecycle-v1
<task_id>:transition:EXECUTING:lifecycle-v1
<task_id>:transition:VALIDATING:lifecycle-v1
<task_id>:transition:COMPLETED:lifecycle-v1
```

相同 OperationKey 的重复 Transition Activity 必须返回之前已提交 / 等价的结果，或者安全的 Already-applied Result，绝不能追加第二次业务 Transition。

S3C Test Fake 必须模拟这条规则，使 Workflow 面向未来 S3D 的真实 Contract 开发，而不是面向更弱的 Test-only Behavior。

## 11. 安全边界

Workflow Input 之所以可信，仅因为它来自 S3B Durable Engine / Outbox Path；它本身仍不是 Authorization Decision。

每一个真实 Business Activity Implementation 在 Repository Access 前都必须重新建立 Tenant/Project Execution Scope。

S3C Fake Activity 验证 Scope Identity，但不授予任何生产 Capability。

任何 Activity Payload 都不携带长期 Credential。

Worker Startup Config 可以包含 Temporal Connection / Auth Configuration，但这些值属于 Process Configuration，绝不能复制进 Workflow History。

## 12. 失败与恢复模型

S3C 必须证明：

- Workflow Open 时 Worker Process Restart；
- Transient Failure 后 Activity Retry；
- Duplicate Workflow Start 已由 S3B Identity Semantic 处理；
- Workflow 开始前 Task 已终态；
- Lifecycle Step 之间 Task 进入终态；
- Transition Seam 返回 Stale ExpectedVersion；
- Worker Graceful Shutdown；
- Replay Test 检测 Workflow Non-determinism。

要求：

1. Worker Restart 不得把 Business Workflow 从零重新开始；Temporal Replay History 后继续编排。
2. Workflow Replay 不得重新执行已经记录完成的 Stub Activity Effect。
3. 观察到终态 Task 后返回成功的 Short-Circuit Result，不产生第二次终态 Mutation。
4. Stale Version 不得 Blind Overwrite；Activity / Workflow 按冻结契约 Reload 或返回 Retryable Conflict Path。
5. Workflow Panic / Non-determinism 不能被 Workflow Layer 伪装成 Business Task Failure。

## 13. 可观测性

Worker / Workflow Log 携带：

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
activity_type
activity_id
```

Log 不记录 Credential 或无界原始 Task / Model Payload。

默认继续抑制 Replay Log，避免 Operational Log 重复。

S3C 不新增第二套 Business Event Stream。真实 Transition Activity 接入后，TaskEvent 仍然是业务事件历史。

## 14. 测试策略

S3C 要求四层测试。

### A. 纯契约测试

- Input Validation；
- Stable Workflow / Activity Name；
- Deterministic Transition OperationKey；
- Terminal-state Recognition；
- Config Default / Fail-closed Behavior。

### B. Temporal Workflow Test Environment

使用 Temporal Go SDK Test Environment + Fake Activities：

- Happy Lifecycle Sequence；
- Expected Activity Order；
- Stub Activity Retry；
- Terminal-before-start Short-Circuit；
- Terminal-between-steps Short-Circuit；
- Stale-version / Reload Path；
- 适用时的 Cancellation Propagation。

### C. Worker Registration / Lifecycle Test

- Workflow 以 `task-execution-v1` 注册；
- Custom Activity Name 显式注册；
- Registration Aliasing Disabled；
- Disabled Config 不执行 Temporal Dial；
- Invalid Enabled Config Fail Closed；
- Graceful Start / Stop Lifecycle 有界。

### D. Replay Compatibility Gate

第一份稳定 S3C Workflow History 必须成为 Checked-in Non-secret Test Fixture 或等价 Generated Fixture。`worker.WorkflowReplayer` 必须能够使用当前 Workflow Implementation 成功重放。

未来 Workflow Change 必须保持 Replay Gate Green，或者显式引入经过 Review 的 Compatibility / Versioning Change。

Repository 常规 Gate 保持：

```text
bilingual docs
go mod tidy
gofmt
go test ./...
PostgreSQL integration tests
go vet ./...
entrypoint builds
```

## 15. 验收标准

S3C 完成条件：

1. `cmd/worker` 从 Skeleton Loop 变为显式 Temporal Enabled / Disabled Lifecycle。
2. Disabled Mode 为默认值且不 Dial Temporal。
3. Worker Registration 使用稳定 External Name 并关闭 Alias Ambiguity。
4. `task-execution-v1` 是唯一主 Task Workflow Type。
5. Workflow Input 只携带有界 Correlation Identity / Schema Version。
6. Workflow 不直接调用任何 External / Database / Provider / Tool。
7. Lifecycle State Change 只能经 Activity Seam 请求。
8. Fake Activity 模拟幂等 Transition Semantic，不产生真实外部 Side Effect。
9. Task Terminal Observation 安全 Short-Circuit。
10. Retry Test 证明重复 Activity Attempt 不等于重复 Business Effect。
11. Workflow Test Environment 与 Replay Compatibility Gate 通过。
12. 尚未激活生产 API-to-Temporal Dispatch Path。
13. 不引入 Deprecated Worker Versioning API。
14. Full CI Pass。

## 16. 回滚策略

S3C 仍具备较安全的 Rollback 条件，因为生产 Task 自动 Dispatch 到 Temporal 仍未激活。

如果 Worker Code 回滚：

- 新 API Task 继续安全留在 PostgreSQL / Outbox；
- 不会静默切换到另一个 Executor；
- 已启动的开发 / 测试 Workflow 由 Replay-compatible Worker Version 处理，或在非生产环境显式终止；
- Rollback 不能为同一 Task 创建第二个 Executor。

生产 Outbox -> Temporal Worker 激活、PostgreSQL-backed Activity、Cancellation Reconciliation 与 Cross-restart Recovery 明确留到 S3D / S3F Gate。

## 17. S3C 实现顺序

```text
C1 Temporal Config + Worker Lifecycle
 -> C2 Stable Workflow/Activity Registration
 -> C3 Deterministic Task Lifecycle Workflow
 -> C4 In-memory Idempotent Fake Activities
 -> C5 Temporal Testsuite Lifecycle/Retry/Terminal Tests
 -> C6 Replay Compatibility Fixture/Gate
 -> C7 Final Architecture/Security Review
```

只有 S3C 通过后，S3D 才把 Fake Business-state Seam 替换为 PostgreSQL-backed Idempotent Activity，并接入 Durable Cancellation / Retry / Reconciliation 行为。
