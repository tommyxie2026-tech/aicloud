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

## 当前研究

| 模型 | 状态 | 观察日期 | 中文入口 | English entry |
|---|---|---:|---|---|
| Kimi K3 | 初始架构研究，已建立双语结构 | 2026-07-29 | [打开中文研究](kimi-k3/README.zh-CN.md) | [Open English study](kimi-k3/README.md) |

## 建议的后续结构

```text
models/
├── kimi-k3/
├── deepseek-*/
├── glm-*/
├── qwen-*/
├── mistral-*/
└── commercial-api-models/
```

模型研究不代表生产批准。生产准入必须在 AI Cloud Model Registry 中完成精确版本的许可证、来源、安全、评测、部署和责任归属审查。
