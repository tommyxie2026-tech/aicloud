# S3B 确定性 Temporal 适配器

## 1. 目标

在暂不激活生产 Task 生命周期 Workflow 的前提下，通过 S3 已冻结的持久工作流边界引入第一层真实 Temporal 集成。

S3B 建立四项基础能力：

1. 当前可维护的 Go / Temporal SDK 基线；
2. 显式携带可信 Tenant/Project/Task/Trace 身份的 Temporal-neutral `DurableEngine` 边界；
3. 与公共 API 幂等键解耦的确定性 Task Workflow Identity；
4. 通过 Outbox 将 Workflow Start 投递到真实 Temporal Client Adapter。

核心归属契约保持不变：

```text
PostgreSQL  = 业务 / 查询状态
TaskEvent   = 追加式业务历史
Outbox      = 事务性投递意图
Temporal    = 持久编排 / 执行历史
```

## 2. 非目标

S3B 不做以下事情：

- 不在 `cmd/worker` 中注册或运行 Task Lifecycle Workflow；
- 不执行 Planning、Routing、Model、Policy、Approval、Validation 或 Tool Activity；
- 不新增公共 HTTP API；
- 不让 Temporal 成为业务事实源；
- 不绕过 TaskEvent、Outbox、Tenant Ownership 或 PostgreSQL RLS；
- 不宣称 Exactly-Once Execution；
- 暂不删除开发兼容用的旧 `workflow.Engine` Seam；
- 不引入真实 Tool 或基础设施副作用。

Worker 注册与确定性生命周期执行属于 S3C。

## 3. 领域模型变更

不新增业务 Aggregate，也不新增 Task State。

S3 执行身份冻结为：

```text
workflow_type = task-execution-v1
workflow_id   = task/<task_id>
```

新的 Temporal-neutral Durable Boundary：

```go
type DurableEngine interface {
    Start(context.Context, StartRequest) (StartResult, error)
    Cancel(context.Context, CancelRequest) error
}
```

`StartRequest` 显式携带可信的 `tenant_id`、`project_id`、`task_id`、`trace_id` 与 Workflow Type。`StartResult.RunID` 仅属于执行证据；Task 仍然是业务身份。

现有 `Engine.Start(ctx, taskID)` 暂时作为 Legacy 非持久 `CreateTask` 的 Deprecated Compatibility Seam 保留。S3C 在生产 Task Start 全部进入 Durable Outbox Path 后必须删除该 Seam。

## 4. API 变更

无。

R7 OpenAPI 契约保持不变。公共调用方不能选择 Temporal Workflow ID，也不能通过 Task Body 提交 Tenant/Project 执行 Scope。

## 5. 数据模型变更

S3B 不需要新增数据库表。

Durable Task Create 的 Outbox Message 做两项加固：

- TaskCreated Payload 增加 `traceId`，使 Delivery Adapter 能从持久数据传播 Correlation，而不是信任 Transport Header；
- `workflow.start` Delivery Identity 改为 Tenant/Task Scope：

```text
workflow-start:<tenant_id>:<task_id>
```

公共 API `Idempotency-Key` 仍然只存入 Command Idempotency Record，并继续表达 API Business Command Replay 语义。

## 6. 安全边界

`workflow.start` Delivery Adapter 只从可信持久 Outbox Field 与已提交 TaskCreated Payload 构建 `StartRequest`。

它验证：

- Destination 必须是 `workflow.start`；
- Tenant、Project、Task Identity 必须存在；
- Aggregate Type 必须为 `Task`；
- Aggregate ID 必须等于 Task ID；
- Payload 中 Task ID 必须等于持久 Outbox Task ID；
- Trace ID 通过 Durable Start Request 传播。

Adapter 不从 HTTP Header 或 Outbox Idempotency 字符串推导业务身份。

`DurableEngine` 不暴露数据库、凭证、Provider、Tool Gateway、Shell、Kubernetes 或基础设施能力。

## 7. 运行时流程

S3B 实现并冻结以下启动路径：

```text
POST /api/v1/tasks
    -> R6 Task Command Transaction
        -> Task
        -> TaskCreated
        -> workflow.start Outbox
        -> API Idempotency Result
    -> COMMIT

Outbox Dispatcher
    -> workflow.StartDeliveryAdapter
    -> DurableEngine.Start(StartRequest)
    -> TemporalEngine
    -> client.ExecuteWorkflow(
           ID = task/<task_id>,
           TaskQueue = configured queue,
           WorkflowType = task-execution-v1)
```

本切片不注册 Worker，因此单独合并 S3B 不会激活生产 Workflow Execution。

## 8. 失败模型

S3B 必须保证 Workflow Start 在 At-Least-Once Outbox Delivery 下安全。

要求：

- 持久身份非法时 Delivery 失败，并保留 Retry / Dead-letter 可见性；
- Temporal 不可用时返回错误，由 Outbox Retry Policy 继续承担投递重试；
- Temporal Start 成功但 Outbox Ack 失败时，重复投递必须安全；
- 同一个确定性 Task Workflow 已启动时，归一为幂等成功；
- 已不存在的 Temporal Execution 被再次 Cancel 时，归一为编排层幂等成功；
- 其他 Temporal Error 必须传播，不能静默吞掉。

S3B 不根据 Temporal Start / Cancel Response 修改 Task Business State。

## 9. 幂等模型

S3B 明确区分三类 Identity：

```text
API Command Key      = 客户端 Idempotency-Key
Outbox Delivery Key  = workflow-start:<tenant_id>:<task_id>
Temporal Workflow ID = task/<task_id>
```

公共 API Key 绝不能成为 Temporal Workflow ID。

Temporal Start 使用确定性 Workflow ID，并拒绝 Duplicate Reuse。SDK Adapter 将相同确定性身份的 `WorkflowExecutionAlreadyStarted` 归一为：

```text
AlreadyStarted = true
```

这是下游持久去重，不是 Exactly-Once 保证。

Activity Idempotency 属于后续 S3C / S3D。

## 10. 可观测性

S3B 传播：

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
workflow_type
```

`workflow_run_id` 由 Temporal 返回，只属于诊断 / 执行证据。

公共 `request_id` 继续存在于现有 TaskEvent / Idempotency Evidence 中，但不会用作 Workflow Identity。

## 11. 成本模型

本切片不引入新的可计费 Model 或 Tool Operation。

S3B 暂不新增 Temporal Orchestration Cost Accounting。未来平台编排成本仍必须可归因到 Task / Trace，并与 Model / Tool Cost Evidence 分离。

## 12. 迁移策略

S3B 将项目语言 / 运行时基线从 Go 1.22 升级到 Go 1.24，并引入：

```text
go.temporal.io/sdk v1.44.1
go.temporal.io/api v1.62.12
```

Repository CI 与 Docker Build Image 在同一切片同步切换到 Go 1.24，避免 Local / Build / Runtime Contract 漂移。

Durable Execution Seam 分阶段迁移：

```text
legacy Engine（开发兼容）
        +
DurableEngine（新的 S3 边界）
        -> S3C 让 DurableEngine 成为权威执行入口
        -> 删除 legacy Engine
```

历史 Outbox Message 的 IdempotencyKey 不能被解释为 Temporal ID。即使旧 Message 的 Key 来源于 API Key，也必须通过 `task/<task_id>` 在下游去重。

## 13. 测试

S3B Merge Gate 包括：

- 确定性 `task/<task_id>` 生成；
- 空 Task Identity 拒绝；
- 可信 Tenant/Project/Task/Trace Validation；
- Outbox Adapter 不使用 Legacy / Public Idempotency-Key 作为 Workflow Identity；
- Outbox Aggregate / Payload Identity Mismatch 拒绝；
- Temporal Start 获得确定性 Workflow ID 与配置 Task Queue；
- AlreadyStarted 归一化；
- 确定性 Cancellation Identity；
- Task Create Outbox Key / Payload Regression Coverage；
- Go 1.24 下 `go mod tidy` Clean；
- 中英文文档一致性检查；
- `gofmt`；
- `go test ./...`；
- PostgreSQL Migration / Task / Outbox Integration Test；
- `go vet ./...`；
- Entrypoint Build。

## 14. 验收标准

S3B 完成条件：

1. Go 1.24 与选定 Temporal SDK 在 CI 中完整构建通过。
2. `DurableEngine` 保持 Temporal-neutral，并显式携带可信 Scope。
3. Workflow ID 对 Task 确定，且与 API / Outbox Key 解耦。
4. Durable Task Create 产生 Task-scoped Workflow Start Delivery Key。
5. Outbox Delivery 能进入真实 Temporal Client Adapter。
6. Duplicate Temporal Start 被安全归一化，但不会隐藏无关错误。
7. Cancellation 使用确定性 Task Workflow Identity，但不修改 Task Business State。
8. 公共 API Surface 无变化。
9. 本切片尚未激活 Task Lifecycle Workflow 或真实 Side Effect。
10. Full CI Pass。

## 15. 回滚策略

S3B 本身尚未激活 Temporal Worker / Lifecycle Workflow，因此具备较安全的行为回滚条件。

在 S3C 激活前，可以移除新 Adapter 并恢复之前的 Build Baseline。S3C 一旦开始真实执行 Task Workflow，Rollback 就必须保证 Single-Executor Ownership 与 Replay Compatibility，而不能仅依赖 Binary Revert。

## 下一切片

S3C 将注册 Temporal Worker，并先使用 Fake / Idempotent Activities 实现确定性 Task Lifecycle Workflow，验证 Worker Restart、Workflow Replay、Task Terminal Short-Circuit 与生命周期编排，再接入 Model / Tool 副作用。
