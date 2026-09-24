# AI Execution OS 与 Agent Execution Provider 集成设计

- 项目：aicloud
- 日期：2026-09-24
- 状态：目标设计 / 迭代输入

## 1. 产品边界

aicloud 的核心问题是：

> 一个 Goal 如何组织 Model、Agent、Tool、Knowledge 与 Runtime，最终形成 Verified Outcome。

主链路：

~~~text
Goal → Planner → Execution Graph → Intelligence Router
     → Execution Provider → Result → Verifier
     → Verified Outcome → Evaluation / Experience
~~~

aicloud 不负责远程 Worker lease、Agent 进程监督、Attempt fencing 等分布式 Agent Job Executor 内部机制。

## 2. 与 computecloud 的关系

computecloud 是独立的 Agent Job Executor。aicloud 可以把它作为一种 **Execution Provider** 使用，但不依赖它才能成立。

~~~mermaid
flowchart TB
  U["User / App"] --> A["aicloud<br/>Goal / Planner / Router / Context / Policy"]
  A --> E["Execution Provider Interface"]
  E --> L["Local / Container / K8s Provider"]
  E --> C["Computecloud Provider"]
  E --> O["Other Provider"]
  C --> R["Execution Result / Events / Artifacts"]
  L --> R
  O --> R
  R --> V["aicloud Verifier"]
  V --> OUT["Verified Outcome"]
  OUT --> F["Evaluation / Experience / Learning"]
  F -.-> A
~~~

原则：

- 两个产品独立部署、独立版本、独立数据模型；
- 不共享数据库；
- aicloud 不感知 computecloud 的 worker_id、lease、generation 等内部字段；
- computecloud 不理解 Goal、Planner、Intelligence Router、Learning Loop；
- 集成关系通过版本化 Execution Contract 建立。

## 3. 两级调度

### L1：aicloud Intelligence Scheduling

决定“用什么智能策略完成任务”：

~~~text
Task × Context × Capability × Quality × Cost × Risk × Policy
    → Model / Agent / Tool / Skill / Execution Provider
~~~

### L2：Execution Provider Scheduling

若选择 computecloud，由 computecloud 决定“在哪里、由谁、以什么 Runtime 执行”：

~~~text
Agent Job
  → Runtime Capability
  → Credential / Workspace / Repository Affinity
  → Worker / Attempt
~~~

aicloud 不复制 L2 调度逻辑。

## 4. Execution Contract v1

建议 aicloud 内部稳定接口：

~~~go
type ExecutionProvider interface {
    Submit(ctx context.Context, req ExecutionRequest) (ExecutionRef, error)
    Get(ctx context.Context, ref ExecutionRef) (ExecutionStatus, error)
    Cancel(ctx context.Context, ref ExecutionRef) error
    Watch(ctx context.Context, ref ExecutionRef) (<-chan ExecutionEvent, error)
}
~~~

Provider：

~~~text
ExecutionProvider
├── LocalProvider
├── ContainerProvider
├── KubernetesProvider
├── ComputecloudProvider
└── OtherProvider
~~~

ExecutionRequest 不泄漏具体 Provider 内部实现：

~~~yaml
apiVersion: execution.aicloud.io/v1alpha1
kind: ExecutionRequest
metadata:
  executionId: exec-123
spec:
  workload:
    type: agent
  capability:
    required: [coding, git, shell]
  input:
    goal: "Fix issue #123"
  context:
    artifacts: [artifact://context/123]
    workspace: workspace://task/456
  constraints:
    timeout: 30m
    isolation: sandbox
  policy:
    permissions: [git.read, git.write, shell.test]
~~~

## 5. 状态语义

Provider 的执行成功不等于 Goal 成功。

~~~text
Execution Provider COMPLETED
        ↓
Result / Artifact
        ↓
aicloud Verifier
   ├─ PASS → Verified Outcome
   └─ FAIL → Replan / Retry / Alternative Provider
~~~

因此必须分离：

~~~text
ExecutionStatus != OutcomeStatus
~~~

标准 Provider Event 建议：

~~~text
QUEUED
ASSIGNED
STARTED
PROGRESS
ARTIFACT_CREATED
WAITING_INPUT
COMPLETED
FAILED
CANCELLED
~~~

## 6. Context 边界

aicloud 负责“给 Agent 什么”：

~~~text
User + Task + Memory + Knowledge + Repository
+ Previous Trajectory + Policy
        ↓
Context Compiler
        ↓
Context / Artifact References
~~~

Execution Provider 负责 materialize 和执行，不负责重新解释知识语义。

## 7. Trajectory 回流

Execution Provider 回传执行事实：

~~~text
duration / retry / runtime / tool calls / failures
artifacts / token usage / resource usage / exit status
~~~

aicloud 增加上层语义：

~~~text
Goal / Plan / Strategy / Evaluation / Outcome / Cost
~~~

组合形成 Execution Trajectory，用于 Evaluation、Router、Planner、Policy 和后续 Learning。

## 8. 迭代计划

### E0 — Contract Definition
- 定义 ExecutionRequest / ExecutionRef / ExecutionEvent / ExecutionResult；
- 明确 ExecutionStatus 与 OutcomeStatus；
- Provider conformance tests；
- 保持 v1alpha1 可演进。

### E1 — Provider SPI
- 在 aicloud 建立 ExecutionProvider；
- LocalProvider 作为 reference implementation；
- Provider registry / capability discovery；
- timeout / cancel / watch。

### E2 — Computecloud Adapter
- 实现 ComputecloudProvider；
- ExecutionRequest → Agent Job 映射；
- Event / Artifact / cancellation 映射；
- 不引入 computecloud 内部领域对象。

### E3 — Verified Execution
- Verifier 接入；
- provider completed → verification；
- verification failed → retry/replan；
- outcome provenance。

### E4 — Routing & Policy
- Provider capability / health / cost metadata；
- Router 可选择 Local/K8s/computecloud；
- policy-aware provider selection；
- fallback。

### E5 — Experience Loop
- 标准化 trajectory；
- Outcome 与 Execution 关联；
- Evaluation 数据回流；
- 优化 Router / Planner / Context / Policy。

## 9. 架构不变量

1. aicloud 可以在没有 computecloud 时运行。
2. computecloud 不进入 aicloud 领域模型。
3. Execution Contract 是唯一正式结合点。
4. Provider COMPLETED 不等于 Verified Outcome。
5. Context semantics 属于 aicloud；runtime execution 属于 Provider。
6. 两级调度不得职责重叠。
7. 任何 Provider 都必须通过同一 conformance contract。
