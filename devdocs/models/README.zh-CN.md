# 模型架构研究

[English](README.md) | **简体中文**

本目录用于保存可能接入 AI Cloud 的商业模型、开放权重模型和自托管模型的工程研究。

研究目的不是复述厂商宣传，而是系统分析：

- 模型架构；
- 模态与上下文行为；
- 预训练和后训练披露；
- 推理与部署架构；
- Benchmark 方法；
- 许可证与供应链证据；
- 运行风险；
- 与 AI Cloud Model Registry、路由、评测、FinOps、Tool Gateway 和 Sandbox 的兼容性。

## 双语规则

每个模型研究目录应至少包含：

```text
README.md          英文入口
README.zh-CN.md    中文入口
<topic>.md         英文章节
<topic>.zh-CN.md   中文对应章节
```

两种语言必须保持相同的章节编号、结论边界、引用和代码示例。

## 对比入口

- [开放模型研究矩阵](comparisons/open-model-research-matrix.zh-CN.md)
- [Open Model Research Matrix](comparisons/open-model-research-matrix.md)

矩阵统一比较架构、数据、训练代码、Checkpoint、后训练、许可证、部署门槛及建议的 AI Cloud 研究角色。

## 当前研究

|模型或项目|研究角色|状态|观察日期|中文入口|English entry|
|---|---|---|---:|---|---|
|Kimi K3|前沿稀疏多模态与长上下文系统|详细双语架构研究|2026-07-30|[打开中文研究](kimi-k3/README.zh-CN.md)|[Open English study](kimi-k3/README.md)|
|OLMo 3 / OLMo 3.1|端到端开放 Model Flow|**详细多章节双语研究**|2026-07-30|[打开中文研究](olmo-3/README.zh-CN.md)|[Open English study](olmo-3/README.md)|
|Apertus|开放多语言与合规导向模型工程|双语技术研究|2026-07-30|[打开中文研究](apertus/README.zh-CN.md)|[Open English study](apertus/README.md)|
|Pythia|跨规模预训练动力学与可解释性|双语技术研究|2026-07-30|[打开中文研究](pythia/README.zh-CN.md)|[Open English study](pythia/README.md)|
|LLM360 Amber|密集 Checkpoint 与数据序列 Trace|双语技术研究|2026-07-30|[打开中文研究](llm360-amber/README.zh-CN.md)|[Open English study](llm360-amber/README.md)|
|Tülu 3|现代开放 SFT、DPO 与 RLVR|双语技术研究|2026-07-30|[打开中文研究](tulu-3/README.zh-CN.md)|[Open English study](tulu-3/README.md)|
|Marin|开放模型实验室运营与实验来源|双语技术研究|2026-07-30|[打开中文研究](marin/README.zh-CN.md)|[Open English study](marin/README.md)|

## 研究分类

```text
前沿架构与 Serving
└── Kimi K3

端到端开放 Model Flow
├── OLMo 3 / 3.1
└── Apertus

预训练动力学
├── Pythia
└── LLM360 Amber

后训练
└── Tülu 3

开放研发运营
└── Marin
```

## 晋升边界

`devdocs/models/` 保存面向实现的外部技术研究。研究结论只有经过评审并迁移到 `docs/` 下的 ADR、设计文档、API 规范、生产策略或路线图后，才会成为稳定的 AI Cloud 技术承诺。

模型研究不代表生产批准。生产准入必须在 AI Cloud Model Registry 中完成精确版本的许可证、来源、安全、评测、部署、容量、成本和责任归属审查。