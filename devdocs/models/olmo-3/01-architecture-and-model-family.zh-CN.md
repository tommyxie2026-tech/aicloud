# 01. 架构与模型家族

[English](01-architecture-and-model-family.md) | **简体中文**

## 1. 家族拓扑

OLMo 3 将基础模型规模与后训练用途分开管理。

```text
OLMo 3 7B Base
├── 7B Instruct
├── 7B Think
└── RL-Zero 研究分支

OLMo 3 32B Base
├── 32B Think
├── OLMo 3.1 32B Think
└── OLMo 3.1 32B Instruct
```

一个最终模型名称至少隐含：

- 基础参数规模；
- 预训练谱系；
- 后训练路径；
- 后训练时长或代际；
- Checkpoint Revision。

## 2. 核心架构

OLMo 3 使用 decoder-only Transformer。官方模型卡披露的稳定维度如下。

|属性|7B|32B|
|---|---:|---:|
|层数|32|64|
|隐藏维度|4096|5120|
|Query Heads|32|40|
|KV Heads|32|8|
|上下文长度|65,536|65,536|
|预训练 Token|5.93T|5.50T|

32B 模型采用 40 个 Query Head 和 8 个 KV Head，表现为 Grouped-Query Attention 结构，相比 Query/KV 一一对应的设计可以降低 KV Cache 增长。

## 3. 架构解释

OLMo 3 的主要差异并不是一个全新的稀疏架构，而是以下系统能力的协同：

```text
常规 Dense Transformer
+ 精细分阶段数据课程
+ 长上下文扩展
+ 开放后训练分支
+ 密集 Checkpoint 证据
+ 可复现评测工具
```

其核心思想是：模型能力是完整系统的结果，而不只是网络结构的结果。

## 4. Base、Instruct 与 Think 是不同产品

### Base

Base Checkpoint 适合继续训练、领域适配、可解释性研究和受控后训练实验。没有明确 Prompt 与安全设计时，不应将它直接作为聊天模型路由。

### Instruct

Instruct 路径侧重通用指令遵循、对话可用性、工具任务和受控回复行为。

### Think

Think 路径通过更长推理轨迹提升数学、编码、逻辑和多步骤任务。其运行特征与 Instruct 不同：

- 输出序列更长；
- 延迟波动更大；
- 每个成功任务成本更高；
- 对 Token Budget 和停止规则更敏感；
- 保存推理轨迹时存在额外隐私要求。

## 5. OLMo 3.1 属于后训练演进

OLMo 3.1 不应被表达为无关的基础架构。官方发布资料说明：

- 32B Think 是对此前最强 RL 运行的延长训练；
- 32B Instruct 是将 Instruct 配方扩展到 32B；
- 两者仍继承 OLMo 3 32B Base 谱系。

AI Cloud 应将其表示为谱系边，而不是把所有名称扁平化成互不相关的模型。

## 6. 推荐 Registry 关系

```yaml
modelFamily: olmo-3
baseModel:
  id: allenai/Olmo-3-1125-32B
  stage: long-context-base
variants:
  - id: allenai/Olmo-3-32B-Think
    pathway: think
    parentStage: rlvr
  - id: allenai/Olmo-3.1-32B-Think
    pathway: think
    parent: allenai/Olmo-3-32B-Think
    changeType: extended-rl
  - id: allenai/Olmo-3.1-32B-Instruct
    pathway: instruct
    parent: allenai/Olmo-3-32B
    changeType: full-32b-instruct-recipe
```

每个 ID 必须同时保存精确仓库 Revision。