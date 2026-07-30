# 04. 后训练：SFT、DPO 与 RLVR

[English](04-posttraining-sft-dpo-rlvr.md) | **简体中文**

## 1. 后训练分支结构

OLMo 3 将后训练公开为多个可检查阶段，而不是只发布一个最终聊天 Checkpoint。

```text
Long-context Base
        ↓
SFT Checkpoint
        ↓
DPO Checkpoint
        ↓
RLVR Checkpoint
        ↓
Instruct 或 Think 最终模型
```

Instruct 与 Think 使用相同的阶段术语，但数据混合、优化目标、输出行为和评测重点不同。

## 2. 监督微调

SFT 通过精选样本教会模型回复格式、指令遵循、推理示范、工具规范和对话行为。

重要证据包括：

- 精确 SFT 混合 Revision；
- 样本格式与 Chat Template；
- 序列长度策略；
- 样本权重；
- Loss Mask；
- Learning Rate 与 Epoch 配置；
- 父级 Base Checkpoint。

即使模型架构没有变化，SFT 也会显著改变模型行为。

## 3. Direct Preference Optimization

DPO 使用偏好回复与拒绝回复塑造模型行为，不必建立独立在线 RL 循环。

其影响可能包括：

- 提升指令遵循；
- 改变回复风格；
- 增强偏好对齐；
- 因偏好数据偏差导致能力退化；
- 改变详细程度和拒绝行为。

AI Cloud 不能假设 DPO Checkpoint 与其 SFT 父模型具有相同的安全和成本特征。

## 4. 可验证奖励强化学习

RLVR 特别适合可以由确定性程序或可靠 Verifier 检查正确性的场景，例如：

- 数学；
- 代码执行与测试；
- 约束型指令遵循；
- 结构化输出要求；
- 具有可验证答案的逻辑任务。

基本循环是：

```text
Prompt
→ Model Rollout
→ Verifier 或 Reward Function
→ 接受/拒绝或标量 Reward
→ Policy Update
```

要实现开放复现，需要 Prompt Set、Rollout 配置、Verifier 实现、Reward 聚合、训练代码和中间 Checkpoint。

## 5. Think 路径

Think 路径显式训练长推理行为，其运行影响包括：

- 平均输出 Token 更多；
- 推理深度波动更大；
- 失败尝试成本更高；
- 对最大 Token 限制敏感；
- 中间推理保存时可能泄露敏感信息；
- 需要区分推理文本质量与最终答案质量。

AI Cloud 只应在任务预期价值足以覆盖额外成本和延迟时路由到 Think。

## 6. Instruct 路径

Instruct 路径侧重通用可用性和指令遵循，通常更适合作为以下任务的默认候选：

- 标准对话；
- 提取与转换；
- 普通工具编排；
- 中低复杂度企业任务；
- 具有严格延迟预算的工作流。

这只是路由假设，而不是生产结论，必须通过本地工作负载验证。

## 7. OLMo 3.1 扩展

官方资料说明，OLMo 3.1 32B Think 在最强 OLMo 3 RL 运行基础上，使用 224 张 GPU 额外训练 21 天，并对 Think RL 数据执行更多轮次。它应被表达为 Extended-RL 后代，而不是新的预训练谱系。

OLMo 3.1 32B Instruct 将 Instruct 路径扩展到 32B，并保留对应 SFT、DPO 和最终 RL Checkpoint。

## 8. 推荐 AI Cloud 路由元数据

```yaml
postTraining:
  pathway: think
  stages:
    - sft
    - dpo
    - rlvr
  reasoningTrace: explicit
  expectedOutputTokens: high
  verifierDomains:
    - math
    - code
    - instruction-following
  parentCheckpoint: <pinned-base>
  datasetRevisions:
    sft: <pinned>
    dpo: <pinned>
    rlvr: <pinned>
```

## 9. 必需的阶段对比评测

每个阶段都应与父级比较：

- 任务成功率；
- 校准与幻觉；
- 安全与拒绝行为；
- 输出长度与成本；
- 延迟分布；
- Tool Call 合法性；
- 格式可靠性；
- 按领域划分的回归。

只评测最终模型会掩盖究竟是哪个阶段引入了能力增益或失败。