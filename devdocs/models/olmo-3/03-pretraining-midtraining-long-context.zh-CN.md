# 03. 预训练、中期训练与长上下文

[English](03-pretraining-midtraining-long-context.md) | **简体中文**

## 1. 三阶段基础模型训练

OLMo 3 将基础模型研发拆成三个阶段：

```text
阶段 A：广覆盖大规模预训练
阶段 B：定向中期训练
阶段 C：长上下文扩展
```

这种拆分形成明确的干预点。研究者可以比较广覆盖 Base、能力强化模型和长上下文模型，而不会把所有变化混在一起。

## 2. 阶段 A：广覆盖预训练

初始阶段使用 Dolma 3 混合建立通用语言能力，主要输出包括：

- Base Checkpoint 谱系；
- 训练配置；
- Token 数量证据；
- 可获得时的 Loss 与评测轨迹；
- 数据混合和处理引用。

7B 与 32B 模型报告的总训练 Token 并不相同，因此规模比较必须同时考虑参数量和训练计算量。

## 3. 阶段 B：中期训练

中期训练在更困难、更聚焦能力的数据分布上继续因果语言建模，用于强化数学、科学、代码、指令理解和推理等能力。

其工程逻辑是：

```text
广覆盖通用表示
+ 集中的高价值数据
→ 在指令微调前形成更强基础能力
```

相比把所有能力都交给 SFT 或 RL，这种方式通常更容易形成稳定的基础能力。

## 4. 阶段 C：长上下文扩展

长上下文阶段使用长文档及相关中期训练数据，将模型扩展到 65,536 Token。

长上下文训练影响的不只是最大序列长度，还包括：

- 位置行为；
- Attention 内存使用；
- Sequence Packing；
- 训练 Batch 组成；
- 文档边界处理；
- 长距离召回与推理。

模型能够接收 65K Token，不代表它在所有位置都具有均匀的召回和推理质量。上下文容量与上下文质量应是两个 Registry 字段。

## 5. Checkpoint 边界也是治理边界

每个阶段都应产生不可变 Registry Identity。

```yaml
stages:
  - id: olmo3-32b-pretrain
    stage: broad-pretraining
  - id: olmo3-32b-midtrain
    parent: olmo3-32b-pretrain
    stage: targeted-midtraining
  - id: olmo3-32b-longcontext
    parent: olmo3-32b-midtrain
    stage: long-context-extension
```

这样可以支持：

- 定向回归测试；
- 回滚到最后一个可接受阶段；
- 领域分支；
- 能力变化归因；
- 独立的安全与许可证证据。

## 6. 复现要求

有效复现不能只依赖架构和最终权重，还需要：

- 精确训练代码 Revision；
- 优化器与调度器配置；
- Global Batch 与 Sequence Packing；
- Tokenizer Revision；
- 数据集与混合 Revision；
- 数据顺序或采样策略；
- 精度和分布式策略；
- Checkpoint 转换过程；
- 评测计划和 Harness。

## 7. AI Cloud 评测计划

不同阶段应使用相同工作负载套件比较：

|阶段|必须重点评测|
|---|---|
|广覆盖预训练|知识、语言建模、毒性基线、记忆|
|中期训练|数学、代码、阅读理解、指令敏感度|
|长上下文|按位置召回、多文档综合、延迟、内存、Lost-in-the-middle|

这样才能把开放 Model Flow 转化为可操作证据，而不是只停留在文档层。