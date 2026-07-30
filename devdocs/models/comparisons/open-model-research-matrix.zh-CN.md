# 开放模型研究矩阵

[English](open-model-research-matrix.md) | **简体中文**

**观察日期：** 2026-07-30

## 1. 目的

本矩阵比较各项目公开的工程证据，而不是只比较一个 Benchmark 分数或含义模糊的 `open-source` 标签。

## 2. 技术与开放度矩阵

|项目|架构与规模|数据|训练代码与配方|Checkpoint|后训练|许可证状态|主要 AI Cloud 角色|
|---|---|---|---|---|---|---|---|
|Kimi K3|2.8T Sparse MoE、约 104B 激活参数、原生多模态、约 100 万上下文|有限披露|部分架构与方法披露|侧重最终发布|复现能力有限|带规模条件的自定义许可证|前沿 MoE、多模态、长上下文和复杂 Serving 研究|
|OLMo 3 / 3.1|7B / 32B Dense Decoder-only、65K 上下文|Dolma 3、Dolmino、Longmino、Dolci|OLMo-core、Open Instruct、分阶段配方|主要 Base、中期、长上下文、SFT、DPO、RLVR 阶段|高度开放|本研究检查的最终 Checkpoint 为 Apache-2.0|端到端开放 Model Flow 首要参照|
|Apertus|8B / 70B Dense、多语言、65K 上下文|开放重建与合规流程|发布训练细节和配方|提供中间 Checkpoint|披露 SFT 与 QRPO|本研究检查的 Checkpoint 为 Apache-2.0|多语言、数据权利、合规和主权部署研究|
|Pythia|14M–12B GPT-NeoX 套件|Pile 版本、预 Tokenize 顺序可复现|GPT-NeoX 配置与复现工具|每模型 154 个|不是现代后训练项目|按 Artifact 使用开放研究许可证|跨规模训练动力学与可解释性实验室|
|LLM360 Amber|6.7B LLaMA 风格 Dense、2K 上下文|约 1.259T 完整序列，拆分为 360 Chunk|训练、数据准备、日志和配置|360 个|有限|Apache-2.0|单次预训练 Trace 和数据/Checkpoint 映射|
|Tülu 3|Llama 3.1 8B / 70B / 405B 后训练家族|开放 SFT、Preference、RLVR 混合|Open Instruct 配方与命令|SFT、DPO、最终 RLVR 阶段|主要研究重点，高度开放|Llama 3.1 Community License|现代后训练与 Verifier 研究|
|Marin|持续演进的 Dense 与 MoE 实验，已公开 8B/32B 工作|按实验开放数据 Artifact|实验声明、基础设施、报告与失败记录|取决于具体运行|持续发展 SFT/RL/Distillation|按 Artifact 分别判断|开放模型实验室运营与 Experiment Registry 参照|

## 3. 按研究问题选择项目

|研究问题|优先项目|原因|
|---|---|---|
|如何表达端到端开放 Model Flow？|OLMo 3|清晰的预训练、中期训练、长上下文和后训练谱系|
|开放模型如何结合数据权利与多语言治理？|Apertus|开放/合规数据重建与多语言重点|
|行为如何随规模和训练时间形成？|Pythia|跨多个规模共享数据顺序，并提供 154 个 Checkpoint|
|单次预训练运行每个阶段发生了什么？|Amber|密集 Checkpoint 与数据 Chunk Sequence|
|如何实现和评测 SFT、DPO 与 RLVR？|Tülu 3|开放数据、代码、配方、Verifier 和评测|
|开放模型实验室应如何运行？|Marin|公开假设、PR、执行、指标、失败和后续实验|
|如何服务与治理前沿稀疏多模态模型？|Kimi K3|极端 MoE、多模态、长上下文与 Serving 复杂度|

## 4. 开放层次

```text
Layer 1 — 成品
权重、配置、Tokenizer、推理代码

Layer 2 — 配方
训练数据、数据混合、配置、SFT/DPO/RL 目标

Layer 3 — 实验证据
Checkpoint、日志、数据顺序、评测 Trace、负结果

Layer 4 — 工厂工具
训练框架、编排、数据处理、Verifier 与 Serving System

Layer 5 — 生产运营
凭据、客户流量、事故历史、容量与安全控制面
```

没有项目会完整开放 Layer 5 的所有细节。真正有意义的是 Layers 1–4 能被独立检查和复现到什么程度。

## 5. 推荐 AI Cloud 研究顺序

```text
1. Pythia / Amber
   验证 Checkpoint 谱系、数据映射和评测灵敏度。

2. Tülu 3
   验证 SFT、DPO、RLVR、Verifier 和后训练证据。

3. OLMo 3
   将完整 Model Flow 接入 Registry、Evaluation 与 FinOps。

4. Apertus
   加入多语言、合规、删除和数据权利治理。

5. Marin
   加入 Experiment Registry、负结果保存和开放实验室工作流。

6. Kimi K3
   将治理系统应用于透明度较低、规模更大的 Serving 目标。
```

## 6. Registry 统一维度

```yaml
modelResearch:
  modelFamily: <id>
  exactRevision: <commit-or-tag>
  architectureClass: <dense-or-moe>
  modalities: []
  contextWindow: <tokens>
  openness:
    weights: <level>
    data: <level>
    trainingCode: <level>
    checkpoints: <level>
    postTraining: <level>
    evaluation: <level>
  licenseEvidence: <reference>
  lineageEvidence: []
  deploymentProfiles: []
  unknowns: []
  recommendedUse: research-only
```

## 7. 决策原则

最开放的模型不一定是最好的生产模型，能力最强的模型也不一定是最容易治理的模型。

AI Cloud 应优化：

```text
工作负载成功率
+ 可解释谱系
+ 安全边界
+ 可恢复性
+ 每个成功任务总成本
+ 许可证与来源可信度
+ 运营责任归属
```

所有生产决策都必须针对精确版本。