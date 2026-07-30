# 05. Checkpoint、评测与可复现性

[English](05-checkpoints-evaluation-and-reproducibility.md) | **简体中文**

## 1. Checkpoint 是实验性证据

OLMo 3 发布主要训练阶段的 Checkpoint，而不是只把最终模型视为有价值的 Artifact。

```text
预训练 Checkpoint
→ 中期训练 Checkpoint
→ 长上下文 Checkpoint
→ SFT Checkpoint
→ DPO Checkpoint
→ RLVR Checkpoint
→ 延长训练的 OLMo 3.1 Checkpoint
```

这允许回答因果问题：

- 代码能力是在中期训练还是 RLVR 阶段提高？
- 指令遵循提高时，事实校准是否退化？
- 长上下文扩展是否影响短上下文质量？
- 延长 RL 是否以更高延迟或更长输出为代价提升推理？

## 2. Checkpoint Identity

Checkpoint Identity 应包括：

```yaml
checkpoint:
  modelRepository: allenai/Olmo-3.1-32B-Think
  revision: <immutable-revision>
  stage: rlvr-extended
  parentRevision: <parent>
  trainingCodeCommit: <commit>
  dataRevision: <revision>
  configDigest: <sha256>
  weightManifestDigest: <sha256>
```

Hugging Face Branch 或 Tag 可以用于发现模型，但不可变 Commit 与 Artifact Digest 才是生产权威。

## 3. 评测层次

OLMo 研究使用多种评测工具和任务套件。AI Cloud 应区分：

|层次|目的|
|---|---|
|训练期评测|发现训练发散和能力趋势|
|标准学术 Benchmark|在声明的 Harness 下比较公开 Checkpoint|
|阶段差异评测|将增益和回归归因到一个训练转换|
|系统评测|测量推理服务、工具使用、结构化输出和长上下文行为|
|业务工作负载评测|测量每个成功任务成本和运营价值|
|安全评测|测试滥用、Prompt Injection、数据泄露和工具边界|

## 4. 可复现评测记录

```yaml
evaluationRun:
  modelRevision: <pinned>
  harness:
    name: olmes
    commit: <pinned>
  tasks:
    manifestDigest: <sha256>
  promptTemplateDigest: <sha256>
  generation:
    temperature: 0
    maxTokens: 4096
    reasoningMode: think
  tools:
    enabled: false
  environment:
    imageDigest: <sha256>
    accelerator: <type>
  resultDigest: <sha256>
```

缺少这些配置时，即使 Benchmark 名称相同，两个分数也可能不可比较。

## 5. OLMo 3.1 Benchmark 解释

Ai2 报告延长 RL 后，OLMo 3.1 32B Think 在数学、逻辑、指令遵循、编码和多步骤任务上提高。这支持“更长 RL 仍可继续带来增益”的假设。

这些结果不能证明：

- 企业工作负载获得同等提升；
- 每个成功任务成本下降；
- 安全性提高；
- 测试领域以外事实性提高；
- 使用其他推理栈时结果相同。

## 6. 可复现等级

|等级|证据|
|---|---|
|L0|仅最终权重|
|L1|权重、架构和模型卡|
|L2|训练代码和高层数据说明|
|L3|固定数据、配置、阶段 Checkpoint 和评测 Harness|
|L4|日志、环境、数据顺序、优化器状态和独立重复训练|

OLMo 3 的目标显著高于普通开放权重发布，但每个分支仍需独立评估。`logs coming soon` 或已宣布但尚未验证的 Artifact 必须保持 `pending`。

## 7. 阶段门禁

新的 OLMo Checkpoint 不应自动替换现有生产路由。必须检查：

- 无关键安全回归；
- 工作负载成功率提升或专业化理由充分；
- P95 延迟和内存有界；
- 每个成功任务成本可接受；
- Tokenizer 与 Chat Template 兼容；
- Artifact 和许可证证据已验证；
- 在稳定观察期完成前保留回退路由。

## 8. 研究价值

OLMo 3 对 AI Cloud 的核心价值，是让我们可以先在可见模型谱系上建立评测方法，再将这些方法应用于 Kimi K3 等透明度较低的模型。