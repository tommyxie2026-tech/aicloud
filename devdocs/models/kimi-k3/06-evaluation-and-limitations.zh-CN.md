# 06. 评测与局限

[English](06-evaluation-and-limitations.md) | **简体中文**

## 1. 评测立场

Kimi K3 官方报告覆盖推理、Coding、Agent、知识工作和视觉等大量 Benchmark。这些结果能说明模型的目标能力，但应被解释为：

> 厂商在特定 Harness 和推理配置下报告的测量结果。

它们不自动等同于：

- 独立复现；
- 企业实际工作负载表现；
- 换用不同 Serving Engine 后仍得到相同结果；
- 降低推理强度后仍得到相同结果；
- 不使用工具时仍得到相同结果；
- 具备经济可行性的生产表现。

## 2. 已发布评测配置

官方模型概述显示，Kimi K3 Benchmark 通常采用：

- `reasoning_effort = max`；
- `temperature = 1.0`；
- 部分单步任务 `top_p = 0.95`；
- Agent 任务 `top_p = 1.0`；
- 某些 Benchmark 使用工具增强；
- 使用 Benchmark 特定 Agent Harness；
- 部分多模态任务进行多次运行取平均。

配置是结果的一部分。脱离配置的分数不是完整证据。

## 3. 厂商报告结果的主要领域

官方发布报告在以下领域表现较强：

- 长时程软件工程；
- Terminal 与代码仓库任务；
- Web Research 与浏览；
- MCP 与工具使用；
- Office 和 Spreadsheet 任务；
- 视觉文档与多模态推理；
- 金融、法律和专业知识工作。

报告同时承认，Kimi K3 在总体比较中仍落后于最强的闭源系统，但在不少任务上超过其他参评模型。

本文不复制完整排行榜，精确数值应以官方发布和技术报告为准。

## 4. Harness 依赖

模型表现可能随 Agent Harness 显著变化。

Kimi K3 报告针对不同模型和任务使用不同 Harness。在部分 Coding Benchmark 中，Kimi K3 使用 Kimi Code，而其他模型可能使用 Codex、Claude Code、Terminus 或其他框架。

### 结果含义

Benchmark 实际测量的是一个系统：

```text
基础模型
+ Reasoning Effort
+ System Prompt
+ Agent Harness
+ Tool Schema
+ Retry Policy
+ Context Management
+ Sandbox
+ Evaluator
```

它并不只隔离测量基础模型本身。

## 5. 工具增强与非工具结果

部分评测同时报告：

- Model-Only 或无工具表现；
- 使用 Python 或通用工具增强后的表现。

两者不能合并成一个能力标签。工具增强会增加：

- 执行准确性；
- 检索或计算能力；
- 新的失败模式；
- 额外成本；
- 安全要求；
- 对 Tool Gateway 与 Sandbox 质量的依赖。

AI Cloud 应把工具增强评测作为独立配置和结果保存。

## 6. 长上下文评测

声明 1,048,576 Token 上下文，应分别评测多个属性：

| 属性 | 示例问题 |
|---|---|
| 接受能力 | Endpoint 是否能成功接收该长度上下文？ |
| 检索 | 能否在不同位置找到精确信息？ |
| 多跳推理 | 能否组合远距离证据？ |
| 指令保持 | 很早出现的指令是否仍有效？ |
| 近因偏置 | 是否过度重视最近内容？ |
| Context Compaction | 压缩历史后质量如何变化？ |
| 工具历史稳定性 | 数百次调用后能否保持状态？ |
| 成本与延迟 | 任务是否具有经济可用性？ |

官方报告在部分评测中使用 Context Compaction。使用压缩和不压缩完整上下文的结果应视为两个不同系统配置。

## 7. 推理强度评测

Kimi K3 支持 `low`、`high` 和 `max`，生产评测应覆盖全部等级。

```text
质量
vs.
延迟
vs.
输出 Token
vs.
工具调用
vs.
实际任务成本
```

在 `max` 下领先的模型未必是普通任务的最佳路由。

推荐指标：

- 各 Effort Level 成功率；
- P50/P95 延迟；
- 输出和推理 Token；
- 工具调用次数；
- Retry 次数；
- 每个成功任务成本；
- 人工介入率；
- 安全策略违规率。

## 8. Agent 评测

长时程 Agent 不能只评估最终是否完成。

### 能力指标

- 任务成功率；
- 首个有用工件产生时间；
- 总完成时间；
- 工具选择正确率；
- 工具失败恢复能力；
- 验证质量；
- 工件正确性；
- 回滚质量。

### 可靠性指标

- 重复动作循环；
- 停滞轨迹；
- 无效工具参数；
- 上下文损坏；
- Compaction 后失败；
- 不必要的高成本模型调用；
- Fallback 频率；
- 未完成 Finalization。

### 安全指标

- 未授权工具调用尝试率；
- Prompt Injection 易感性；
- 凭据处理违规；
- Network Policy 违规；
- Workspace Escape 尝试；
- Policy Bypass 尝试；
- Human Approval Bypass 尝试；
- 审计完整性。

## 9. 多模态评测

多模态结果应按任务类型拆分：

- 图像感知；
- Chart 和 Diagram 理解；
- OCR 与文档结构；
- 数学视觉推理；
- 多图比较；
- 视频理解；
- 结合 Python 或其他工具的视觉任务。

公开 Hugging Face 接口和技术报告不一定通过相同 API 契约暴露所有模态。AI Cloud 必须评测精确 Endpoint 与 Processor 组合。

## 10. Coding 评测

对 AI Cloud Coding 场景，Benchmark 成功还不够，评测链应覆盖：

```text
Issue 理解
→ 仓库导航
→ Patch 生成
→ Build 与 Test
→ 安全扫描
→ Policy Check
→ 人工审查
→ PR 质量
```

推荐指标：

- Patch 接受率；
- 在不削弱测试的情况下通过测试；
- 回归率；
- 新增依赖风险；
- Secret 暴露率；
- Changed-Line Efficiency；
- Review Comment Rate；
- 回滚成功率；
- 每个合并变更的成本。

## 11. 可复现评测记录

AI Cloud 应根据以下配置生成稳定 Digest：

```yaml
model:
  id: moonshotai-Kimi-K3
  revision: <pinned-revision>
  engine: <vllm-or-sglang-version>
  endpointMode: self-hosted-or-api
inference:
  reasoningEffort: max
  temperature: 1.0
  topP: 1.0
  maxOutputTokens: 32768
context:
  inputTokens: <measured>
  compactionPolicy: <version>
tools:
  schemaDigest: <digest>
  permissions: <policy-version>
agent:
  harness: <name-and-version>
  maxTurns: <number>
sandbox:
  imageDigest: <digest>
  networkPolicy: deny-by-default
dataset:
  suite: <suite>
  version: <version>
evaluator:
  version: <version>
```

最终结果应同时保存 Configuration Digest 和 Raw Artifact Digest。

## 12. Release Gate

Kimi K3 Revision 不应只因为平均 Benchmark 高就进入生产路由。

建议门禁：

- 最低任务成功率；
- Critical Safety Failure 必须为零；
- 每个成功任务最大成本；
- 按上下文区间设置最大 P95 延迟；
- 最大人工介入率；
- 最大 Retry 和 Fallback 率；
- 不允许许可证或来源准入失败；
- 不允许出现无法解释的前版本回归。

## 13. 公开证据中的已知局限

### 复现不完整

完整数据、训练栈、奖励系统和内部 Benchmark 未发布。

### 厂商报告比较

部分比较使用不同 Agent Harness，或引用不同时间获取的外部排行榜结果。

### 强调最大 Effort

多数头部结果使用最大推理强度，可能掩盖延迟与成本权衡。

### 内部 Benchmark

部分重要能力与安全套件为内部评测，无法独立检查。

### Serving 依赖

表现会随 Engine、内核、量化支持、Context Management 和 Endpoint Revision 改变。

### 长上下文不确定性

最大可接受上下文不保证整个窗口内都具备均匀有效推理能力。

### 开放权重治理

权重公开，但训练数据来源和完整复现能力仍有限。

## 14. 建议的 AI Cloud 评测套件

```text
K3-CAPABILITY-001  结构化推理
K3-CODE-001        从仓库 Issue 到 Patch
K3-LONGCTX-001     1M Context 检索与多跳推理
K3-AGENT-001       长时程工具工作流
K3-VISION-001      文档与 Chart 理解
K3-SAFETY-001      Prompt Injection 与权限边界
K3-COST-001        按 Effort 计算每个成功任务成本
K3-FAILOVER-001    Endpoint 故障与 Fallback
K3-LICENSE-001     证据与准入验证
K3-CACHE-001       混合 Prefix Cache 隔离与删除
```

所有套件必须针对精确生产 Revision 和部署路径运行。
