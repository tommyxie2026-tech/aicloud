# 参考资料

[English](references.md) | **简体中文**

## 一手来源

### 官方仓库

- MoonshotAI/Kimi-K3  
  https://github.com/MoonshotAI/Kimi-K3

用于核查：

- 官方模型介绍；
- 已发布模型规模；
- 部署建议；
- 使用行为；
- Reasoning Effort 字段；
- 保留 Thinking History 指南；
- 许可证链接；
- 技术报告。

### 技术报告

- Kimi K3: Open Frontier Intelligence，arXiv:2607.24653  
  https://arxiv.org/abs/2607.24653  
  https://arxiv.org/pdf/2607.24653

关键章节：

| 主题 | 报告章节 |
|---|---|
| 总体架构 | Section 2 与 Figure 2 |
| Kimi Delta Attention | Section 2.1.1 |
| Gated MLA | Section 2.1.2 |
| Attention Residuals | Section 2.2 |
| Stable LatentMoE | Section 2.3 |
| Native Vision | Section 2.4 |
| 预训练数据与课程 | Section 3 |
| SFT、RL 与策略整合 | Section 4 |
| 面向部署的 QAT | Section 4.1.4 |
| KDA 系统协同设计 | Section 5.1 |
| MoE 预训练基础设施 | Section 5.2 |
| 百万 Token Agentic RL 基础设施 | Section 5.3 |
| 推理和在线 Serving | Section 5.4 |
| 公开评测 | Section 6.1 |
| 内部评测 | Section 6.2 |

### Hugging Face 模型仓库

- moonshotai/Kimi-K3  
  https://huggingface.co/moonshotai/Kimi-K3

用于核查：

- 权重分片清单；
- 仓库体积；
- `config.json`；
- Generation Config；
- 自定义 Transformers 代码；
- Processor 与媒体工具；
- 发布 Checkpoint 元数据；
- 模型卡使用示例。

重要文件包括：

```text
config.json
generation_config.json
configuration_kimi_k3.py
modeling_kimi_k3.py
modeling_kimi_linear.py
kimi_k3_processor.py
kimi_k3_vision_processing.py
media_utils.py
encoding_k3.py
model.safetensors.index.json
model-00001-of-000096.safetensors
...
model-00096-of-000096.safetensors
```

文件清单可能变化；生产记录必须固定仓库 Revision。

### 许可证

- Kimi K3 License  
  https://github.com/MoonshotAI/Kimi-K3/blob/main/LICENSE

用于核查：

- 授予的权利；
- Copyright 与 Notice 要求；
- Model-as-a-Service 定义；
- 连续 12 个月 2000 万美元收入条件；
- 1 亿 MAU 与每月 2000 万美元收入的归属条件；
- 内部使用和认证合作伙伴豁免；
- Warranty 与 Liability Disclaimer。

### 官方 API 与使用文档

- Kimi Platform  
  https://platform.kimi.ai

实施时用于核查当前 API Contract，包括：

- OpenAI-Compatible 接口；
- Anthropic-Compatible 接口；
- `reasoning_effort` 取值；
- 多模态 Message；
- Structured Output；
- Tool Choice 与 Dynamic Tool；
- Context Caching；
- 当前价格、Quota 与 Region。

API 行为和价格具有时效性，必须在实施时重新确认。

## 报告引用的相关开放项目

### AgentENV

- https://github.com/kvcache-ai/AgentENV

技术报告引用 AgentENV 作为面向 Agent 工作负载的 microVM Sandbox。公开该项目并不意味着 Moonshot AI 的完整内部 RL 编排已经开放。

### vLLM

- https://github.com/vllm-project/vllm
- https://docs.vllm.ai

### SGLang

- https://github.com/sgl-project/sglang
- https://docs.sglang.ai

### TokenSpeed

- https://lightseek.org

引擎支持必须针对精确版本和 Kimi K3 Recipe 验证。

## 来源质量层级

按以下顺序使用来源：

1. 精确发布文件与配置；
2. Kimi K3 License；
3. 官方技术报告；
4. 官方模型卡与 API 文档；
5. 推理引擎官方文档；
6. 独立 Benchmark 复现；
7. 可信二手报道；
8. 社区评论和未验证部署声明。

## 后续更新的引用规范

更新文档时应：

- 链接精确来源；
- 标注观察日期；
- 区分厂商声明与独立结果；
- 避免大段复制来源文本；
- 生产结论固定 Model 与 Code Revision；
- 记录报告、模型卡、配置和实际 Runtime 之间的矛盾；
- 将过时结论移入 Change Log，而不是静默改写历史；
- 在同一个 PR 中同步更新英文与中文对应文件。

## 快照说明

本参考清单反映 **2026-07-29** 可获得的公开材料。Kimi K3 刚发布，仓库内容、推理配方、文档和生态支持预计会持续演进。
