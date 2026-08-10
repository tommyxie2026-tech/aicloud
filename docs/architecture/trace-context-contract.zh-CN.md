# AI Cloud Trace 与 Correlation Context 契约

> 状态：S0 Contract Freeze

## 1. 目标

定义稳定 ID 与传播规则，使任意 Task 都可以跨 API、Workflow、Model、Policy、Tool、Sandbox、Audit、Cost、Evaluation 完整重建。

## 2. 标准 Correlation ID

```text
request_id
trace_id
tenant_id
project_id
task_id
workflow_id
agent_id
model_attempt_id
tool_invocation_id
policy_decision_id
approval_id
evaluation_run_id
```

并非每个 Operation 都拥有全部 ID，但只要某个 ID 已存在，就必须在后续链路中一致传播。

## 3. Identity Hierarchy

```text
Request
  └─ Task
      └─ Workflow
          ├─ Agent step
          │   ├─ ModelAttempt
          │   └─ ToolInvocation
          ├─ PolicyDecision
          ├─ Approval
          └─ EvaluationRun
```

`task_id` 是主要 Business Correlation Key；`trace_id` 是主要 Telemetry Correlation Key。

## 4. ID 生成规则

- `request_id` 在 API Boundary 生成，或接受符合 Trusted Gateway 格式的值；
- `trace_id` 遵循 OpenTelemetry/W3C Trace Context；
- `task_id` 在 Task Creation 时生成一次，之后永不改变；
- Attempt/Invocation/Decision ID 是唯一且不可变的 Record Identity；
- Retry 不能因为重新执行就创建新的 Task/Trace Business Identity。

## 5. Context Propagation

Normalized Execution Context 至少携带：

```yaml
context:
  principal: ...
  tenant_id: string
  project_id: string
  request_id: string
  trace_id: string
  task_id: string?
  workflow_id: string?
```

HTTP 使用 W3C `traceparent`/`tracestate`。Workflow/Activity Message 即使需要重新建立 Telemetry Context，也必须显式携带 Business Correlation ID。

## 6. Logging

所有 Task-related Structured Log 至少包含：

```text
timestamp
level
component
request_id
trace_id
tenant_id
project_id
task_id
operation
message
```

默认不能记录敏感 Prompt/Data/Credential 原文。

## 7. OpenTelemetry Span Model

推荐：

```text
HTTP request
  -> task.command
    -> workflow.start/signal
      -> agent.step
        -> router.decide
        -> model.attempt
        -> policy.decide
        -> tool.invoke
          -> sandbox.execute
        -> validation
        -> evaluation
```

Span Attribute 保存调查所需 ID/Version，但大型 Payload 与 Secret 应保存为受治理 Evidence Reference，而不是直接写入 Span。

## 8. Audit 与 Cost Linkage

Task-related AuditEvent 与 CostEvent 必须包含：

```text
tenant_id
project_id
task_id
trace_id 或等价 evidence link
```

从一个 Task/Trace 应能够回答：

```text
发生了什么？
谁触发？
用了哪个 Model/Tool？
为什么被允许？
花了多少钱？
```

## 9. Retry/Fallback

Retry 保持相同 Task 与 Logical Operation Context，但物理 Attempt 可以生成新 ID：

```text
same task_id
same logical operation_id
new model_attempt_id
```

Fallback 默认保持同一 Task/Trace Lineage；如确需新 Trace，必须使用 Trace Link 明确关联。

## 10. Sampling

Security/Audit/Cost Evidence 不受 Telemetry Sampling 影响。Trace Sampling 可以减少低价值 Span，但不能删除 Mandatory Business Evidence。

High-risk Tool Operation 与 Failure 建议 100% Trace Sampling，除非 Data Policy 禁止。

## 11. Retention 与 Privacy

Identifier 的 Retention 必须满足 Audit Correlation；Payload Retention 由 Data Policy 独立控制。Trace/Log 系统必须支持 Redaction 与 Tenant-aware Access Control。

## 12. 验收条件

- 一个 Task ID 可以重建 Route、ModelAttempt、Policy、Approval、Tool、Audit、Cost、Evaluation；
- Request/Trace/Task ID 在 API/Worker Restart 后保持；
- Retry 不创建新的 Task Identity；
- Structured Log 始终带 Correlation Context；
- OpenTelemetry Context 跨 API/Workflow/Activity 传播；
- Trace Sampling 不影响 Mandatory Audit/Cost/Business Evidence；
- Default Trace/Log 中不存在 Secret。