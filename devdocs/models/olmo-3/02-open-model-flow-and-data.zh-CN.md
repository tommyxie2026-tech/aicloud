# 02. 开放 Model Flow 与数据

[English](02-open-model-flow-and-data.md) | **简体中文**

## 1. Model Flow 是主要 Artifact

OLMo 3 将模型表达为一系列可以检查的转换过程。

```text
原始与精选来源
→ Dolma 3 处理和混合
→ 大规模预训练
→ Dolmino 中期训练
→ Longmino 长上下文扩展
→ Base Checkpoint
→ Dolci SFT
→ Dolci DPO
→ Dolci RLVR
→ Instruct / Think 最终 Checkpoint
```

当每条箭头都具备以下证据时，可复现性最强：

- 输入数据集 Revision；
- 转换代码 Revision；
- 训练配置；
- 父、子 Checkpoint ID；
- 指标和评测配置；
- 许可证及来源证据。

## 2. Dolma 3

Dolma 3 是大规模预训练数据基础。其工程价值不仅是能够访问数据，还包括检查数据构建、过滤、混合和后续专业化过程。

AI Cloud 应区分：

```text
原始来源语料
处理后语料
训练混合
采样顺序或采样策略
模型实际消费的 Token Stream
```

这些不是可以互相替代的来源对象。

## 3. Dolmino

Ai2 将 Dolmino 描述为从约 2.2T Token Pool 构建的中期训练混合，其中约采样 100B Token 进行定向训练，重点包括：

- 数学；
- 科学；
- 代码；
- 指令遵循；
- 阅读理解；
- 推理轨迹。

因此，中期训练不是普通微调。它继续执行 next-token training，但有意识地将数据分布转向通用网页预训练中相对不足的能力。

## 4. Longmino

Longmino 使用约 50B Token，并从更大的长文档 Pool 中采样，同时混合中期训练材料，以扩展上下文能力。

其目的不是简单修改配置字段，而是训练模型处理：

- 长报告；
- 日志；
- 多章节文档；
- 分散证据；
- 长距离依赖。

AI Cloud 仍需独立测试完整上下文范围内的召回、推理和位置稳健性。

## 5. Dolci

Dolci 是后训练数据套件，为不同阶段提供独立混合：

- 监督微调；
- 偏好优化；
- 可验证奖励强化学习；
- 推理、工具使用、指令遵循、数学、编码和对话。

按阶段拆分数据非常重要。单一 `trainingData` 字段无法描述完整流程。

## 6. 推荐数据谱系对象

```yaml
dataLineage:
  pretraining:
    dataset: allenai/Dolma3
    revision: <pinned>
    mixtureRevision: <pinned>
  midtraining:
    dataset: allenai/Dolmino
    tokenBudget: 100B
    sourcePoolApprox: 2.2T
  longContext:
    dataset: allenai/Longmino
    tokenBudgetApprox: 50B
    sourcePoolApprox: 639B
  posttraining:
    sft: <dolci-sft-revision>
    dpo: <dolci-dpo-revision>
    rlvr: <dolci-rl-revision>
```

## 7. 数据开放不等于免除治理

公开数据提高了可检查性，但不能消除：

- 上游许可证差异；
- 隐私和个人数据问题；
- 来源删除和 Revision 漂移；
- Benchmark 污染；
- 恶意或低质量样本；
- 区域性数据限制。

AI Cloud 应同时保存厂商公开证据和自己的法律、安全及质量结论。