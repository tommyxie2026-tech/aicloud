# Task Execution Contract 研究草案（已由 ECP Convergence 取代）

[English](task-execution-contract.md) | **简体中文**

> 状态：**Superseded / 历史研究输入**  
> 日期：2026-09-16  
> 当前架构：[Execution Control Plane 收敛分析](../docs/architecture/execution-control-plane-convergence.zh-CN.md)

## 1. 状态说明

本文件最初提出 `TaskDefinition → ExecutionPlan → ExecutionRun` 三对象模型。完成代码级 Gap Analysis 后确认，AI Cloud 已经存在更成熟且受 S0 Frozen Contracts 约束的 canonical contracts：

```text
internal/domain.Task
        ↓
execution.Goal (Task intent snapshot)
        ↓
execution.ExecutionPlan (versioned bounded DAG)
        ↓
execution.Execution
        ↓
execution.ExecutionAttempt
```

因此本文件不再作为实现依据，保留仅用于记录研究演进。

## 2. 被替换的概念

| 早期概念 | Canonical contract | 处理 |
|---|---|---|
| `TaskDefinition` | `internal/domain.Task` | 不创建第二 Task aggregate |
| `ExecutionPlan` 草案 | `execution.ExecutionPlan` | 使用现有 bounded-DAG contract |
| `ExecutionRun` | `execution.Execution` | 使用现有 execution lifecycle |
| 独立 Execution Event Store | `domain.TaskEvent` | 复用 canonical business history |
| 自建 workflow engine | Temporal workflow | 复用现有 durable orchestration |
| `internal/taskexec` | `execution/` | 重复包已删除 |

## 3. 保留的研究结论

以下原则继续有效：Task 是一级业务资源；Provider/Model/Agent 是执行能力；Policy 拥有最终授权权；执行必须具备 durable history、cost/latency/resource/outcome telemetry；Provider、Agent Framework 与 Runtime 应解耦；任务级经济指标应逐步从 `$ / token` 转向 Cost per Successful Task。

## 4. 当前实现路线

后续统一使用 `ECP-Cx` 命名，避免与仓库已有 R1–R7 remediation 冲突：

```text
ECP-C1 Task ↔ Execution Binding
ECP-C2 Plan / Route convergence
ECP-C3 Persistent execution
ECP-C4 Policy / Approval / Budget convergence
ECP-C5 Evaluation / Outcome
ECP-C6 Organization Memory
```

当前实施依据是 `docs/architecture/execution-control-plane-convergence.zh-CN.md` 以及现有 S0 Frozen Contracts。

## 5. 变更控制

如果未来需要改变 Task ownership、TaskEvent source of truth、Temporal ownership 或 `execution/` canonical domain，必须走 architecture issue / ADR / migration review，而不能通过新增平行 package 或平行 schema 隐式改变架构。
