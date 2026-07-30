# 00. 范围、来源与可信度

[English](00-scope-sources-and-confidence.md) | **简体中文**

## 1. 研究问题

本研究分别回答四个问题：

1. OLMo 3 / 3.1 的模型架构是什么？
2. 模型制造流程中哪些阶段可被外部检查？
3. 哪些公开 Artifact 能够支持可复现实验？
4. AI Cloud 应如何表达和治理这条 Model Flow？

## 2. 一手来源

研究优先使用：

- Ai2 OLMo 3 发布文章和 OLMo 3.1 更新；
- OLMo 3 / 3.1 官方 Hugging Face 模型卡；
- 用于分布式训练的 `allenai/OLMo-core`；
- 用于 SFT、DPO 和 RLVR 的 `allenai/open-instruct`；
- Dolma 3、Dolmino、Longmino 和 Dolci 数据仓库；
- OLMES 及官方评测配置；
- 可获得时使用精确 Model、Dataset Revision。

存在官方来源时，不使用二手评论作为架构或开放度事实依据。

## 3. 可信度标签

|标签|含义|
|---|---|
|已确认|官方资料直接陈述，或可从已发布 Artifact 验证|
|强推断|多个官方 Artifact 共同支持，但没有一条官方原句完整概括|
|有限|已经宣布或部分说明，但未验证完整 Artifact 集合|
|未知|未找到足够公开证据|

## 4. 本研究使用的稳定事实

官方模型卡确认：

- 7B 与 32B 基础模型家族；
- decoder-only Transformer 架构；
- 65,536 Token 上下文；
- 分阶段预训练；
- Dolma 3 基础数据、Dolmino 中期训练数据和 Longmino 长上下文数据；
- Dolci 后训练数据集；
- Instruct 与 Think 分支采用 SFT、DPO 和 RLVR；
- 本研究检查的最终 Checkpoint 模型卡采用 Apache-2.0；
- 发布主要阶段 Checkpoint 和相关训练信息。

OLMo 3.1 更新新增了两个 32B 最终分支：Think 与 Instruct。

## 5. 重要限制

“开放”必须拆成多个独立维度：

```text
权重访问
训练代码访问
数据访问
配方透明度
Checkpoint 覆盖
训练日志可用性
评测可复现性
许可证清晰度
生产系统透明度
```

一个项目可以具有很高的科研可复现性，同时仍合理保留集群凭据、内部运营记录、安全事故和生产控制面细节。

## 6. Revision 管理

AI Cloud 研究应固定：

- 模型仓库 Revision；
- 数据集 Revision；
- 训练代码 Commit；
- 后训练代码 Commit；
- 评测 Harness Commit；
- 容器和依赖 Digest。

仅记录 `Olmo-3.1-32B-Think` 这类浮动名称，不足以构成生产证据。