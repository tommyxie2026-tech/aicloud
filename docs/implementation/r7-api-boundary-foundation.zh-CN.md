# R7A API 边界基础

## 状态

这是 S2/R7-API 的首个实现切片，用于在 R6 Task Transaction Kernel 之后开始 API/Auth 契约收敛。

## 目标

在引入生产级 OIDC/JWT 适配器以及 RBAC/ABAC 之前，先把认证机制与业务 Handler 解耦，并建立稳定的 Request/Trace 关联标识和公共错误结构。

## 运行时边界

```text
HTTP Request
   |
   v
Request / Trace Metadata
   |
   v
Principal Verifier
   |
   v
Verified Principal
   |
   v
Tenant / Project Scoped Handler
```

业务 Handler 只消费经过验证的 `identity.Principal`，不直接解析 JWT Claim，也不直接解析身份 Header。

## Principal Verifier 契约

`httpapi.PrincipalVerifier` 是 API 边界认证抽象。实现需要把一次请求解析为经过验证的 `identity.Principal`，否则拒绝请求。

现有 Trusted Ingress Header 机制只以 `TrustedIngressVerifier` 的形式保留，明确作为兼容模式，而不是最终生产认证模型。

外部认证流程不能创建内部系统身份；内部系统身份继续使用已有的显式构造和权限约束。

## 关联标识契约

公共请求携带：

- `X-Request-ID`：单次请求关联标识；
- `X-Trace-ID`：端到端 Trace 关联标识。

合法的调用方标识可以保留；缺失或格式非法时由平台重新生成。规范化后的值会写入请求、响应和 Request Context。

## ErrorEnvelope

认证边界失败开始使用冻结的 OpenAPI 错误结构：

```json
{
  "error": {
    "code": "UNAUTHENTICATED",
    "message": "...",
    "request_id": "req-...",
    "trace_id": "trace-...",
    "retryable": false
  }
}
```

R7A 建立统一 ErrorEnvelope 类型和写入器。将所有历史 Handler 错误路径全部迁移到该结构，仍属于后续 R7-API 收敛范围，本切片不宣称已经全部完成。

## 安全决策

1. 未配置 Verifier 时拒绝公共 API 请求。
2. 外部身份校验失败时拒绝请求。
3. Health/Readiness Endpoint 保持在公共 API 认证边界之外。
4. Task API 继续要求 Project Scope。
5. Trusted Ingress 仅保留为兼容模式；下一切片接入生产 OIDC/JWT。
6. 业务 Handler 不感知认证传输协议。

## 下一切片

R7B 将实现生产级 OIDC/JWT Verifier，包括 Issuer、Audience、时间窗口和签名验证，将已验证 Claim 映射为 Principal，并完成生产 Wiring，使原始身份 Header 不再作为信任来源。
