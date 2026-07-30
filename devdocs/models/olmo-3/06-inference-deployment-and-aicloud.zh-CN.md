# 06. 推理、部署与 AI Cloud

[English](06-inference-deployment-and-aicloud.md) | **简体中文**

## 1. 部署画像

OLMo 3 可以形成多种运行画像：

|画像|典型用途|
|---|---|
|7B Base|研究、继续训练、受控领域适配|
|7B Instruct|低成本对话、提取、工具工作流|
|7B Think|成本受限的推理实验|
|32B Base|更高能力研究与领域适配|
|32B / 3.1 Instruct|通用企业指令遵循|
|32B / 3.1 Think|高价值推理、数学、代码、复杂规划|

正确路由取决于任务价值、延迟、内存、输出 Token Budget 和安全要求，而不只是模型大小。

## 2. 推理栈

官方模型卡支持标准 Transformers 加载。生产环境可以使用优化 Runtime，但每一个 Runtime 都会成为评测 Artifact 的一部分。

```text
模型 Revision
+ Tokenizer Revision
+ Chat Template
+ Runtime Version
+ Kernel Version
+ Quantization Method
+ Container Digest
+ Serving Configuration
```

即使模型 ID 不变，量化或 Runtime 变化也可能改变质量、延迟、内存和数值行为。

## 3. 内存与容量规划

32B BF16 Checkpoint 在计算 KV Cache 和 Runtime 开销之前就需要大量加速器内存。量化可以降低权重内存，但生产容量计算必须包括：

- 模型权重；
- 最大 65K 上下文的 KV Cache；
- 临时 Activation；
- Tensor Parallel 通信；
- Batching 与内存碎片；
- Runtime Reserve；
- 并发请求；
- 较长 Think 输出。

容量应在目标上下文与并发分布下测量，而不是依据单请求 Smoke Test。

## 4. 联合路由

推荐路由维度：

```yaml
routingRequest:
  taskClass: code-reasoning
  modelFamily: olmo-3
  variantPreference:
    - instruct
    - think
  inferenceEffort: medium
  maxContext: 32000
  maxOutputTokens: 4000
  serviceTier: standard
  maxSuccessfulTaskCost: <budget>
```

Router 应联合选择：

- 模型规模；
- Instruct 或 Think 路径；
- 推理与输出预算；
- 量化或全精度 Endpoint；
- 延迟等级；
- 回退链。

## 5. Provider 与 Registry 对象

```yaml
modelVersion:
  id: allenai/Olmo-3.1-32B-Think
  revision: <pinned>
  family: olmo-3
  parameters: 32B
  pathway: think
  contextWindow: 65536
  license: Apache-2.0
  parentModel: allenai/Olmo-3-32B-Think
  trainingEvidence:
    pretraining: <refs>
    midtraining: <refs>
    longContext: <refs>
    sft: <refs>
    dpo: <refs>
    rlvr: <refs>
endpoint:
  runtime: vllm-or-other
  runtimeVersion: <pinned>
  quantization: <none-or-scheme>
  health: healthy
  capacity:
    maxConcurrentRequests: <measured>
    maxContextAtTargetSLO: <measured>
```

## 6. Think 路由控制

Think 模型需要明确控制：

- 最大输出 Token；
- 推理超时；
- Early Stop 与 Verifier 行为；
- Trace 保留策略；
- 敏感推理内容脱敏；
- 重试上限；
- 任务级成本上限。

推理轨迹不应自动向终端用户暴露，也不应无限期保存。

## 7. 回退设计

代表性链路：

```text
OLMo 3.1 32B Think
→ OLMo 3.1 32B Instruct
→ OLMo 3 7B Instruct
→ 商业 API 或确定性工作流
```

容量不足、超时、临时 Runtime 故障和限流可以进入回退；Policy、数据驻留、许可证或安全要求不能被静默绕过。

## 8. FinOps

成本核算应包括：

- Accelerator Seconds；
- 预留容量与实际利用率；
- 输入和输出 Token；
- KV Cache 占用；
- Runtime 启动与模型加载；
- 失败尝试和回退尝试；
- Tool 与 Sandbox 成本；
- 评测成本；
- 人工复核。

首要指标是每个成功任务成本，而不是每个生成 Token 成本。

## 9. 生产准入

开放代码和数据能够提高信心，但生产仍需：

- Artifact Digest 与签名；
- 独立工作负载评测；
- 安全与 Agent 边界测试；
- 许可证与数据集审查；
- 容量和故障测试；
- 可观测性与回滚；
- 明确的运营责任人。