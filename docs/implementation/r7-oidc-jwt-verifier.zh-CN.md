# R7B OIDC/JWT 验证器

## 状态

实施阶段：R7B 认证验证器核心。

本文档在 R7A `PrincipalVerifier` 边界上增加面向生产的 OpenID Connect / JWT 验证器。当前阶段**尚不宣称** API Server 生产接线或 RBAC/ABAC 已经完成。

## 目标

把外部 Bearer Token 转换为经过验证的 `identity.Principal`，同时禁止业务 Handler 直接解析或信任未经验证的 Claim。

```text
Authorization: Bearer <JWT>
        |
        v
OIDCVerifier
        |
        +-- HTTPS Issuer Discovery（可选）
        +-- HTTPS JWKS 获取与缓存
        +-- 签名算法白名单
        +-- RSA 签名验证
        +-- Issuer / Audience 校验
        +-- exp / nbf / iat 校验
        +-- Verified Claim Mapping
        v
identity.Principal
```

## 安全不变量

1. Issuer 和 Audience 必须来自显式配置，不能从传入 Token 中动态学习。
2. Discovery 与 JWKS Endpoint 必须是绝对 HTTPS URL。
3. Discovery 返回的 Issuer 必须与配置的 Issuer 完全一致。
4. `alg=none` 以及不在允许列表中的算法必须拒绝。
5. R7B v0.1 支持 `RS256`、`RS384`、`RS512`，默认只允许 `RS256`。
6. 小于 2048 bit 的 RSA Signing Key 不进入可用 Key Set。
7. JWT 必须包含 `kid`。遇到未知 `kid` 时只允许强制刷新 JWKS 一次，以支持正常 Key Rotation，之后继续 Fail Closed。
8. `exp` 必须存在；`nbf` 和 `iat` 在存在时必须按有限 Clock Skew 校验。
9. Audience 必须精确匹配，同时支持字符串和数组形式的 `aud`。
10. 外部 JWT 只能映射为 `user` 或 `service_account`；任何外部 `system` Principal 都必须拒绝。
11. Tenant、Project、Role、Capability 只有在签名和标准 Claim 验证成功之后才允许进入 Principal。
12. Token 只从唯一的 `Authorization: Bearer ...` Header 读取，不支持 Query String 或 Cookie 降级读取。
13. Bearer Token 有长度上限，并且不得写入日志。

## 默认 Claim 映射

| 平台字段 | JWT Claim |
|---|---|
| `SubjectID` | `sub` |
| `TenantID` | `tenant_id` |
| `ProjectID` | `project_id` |
| `Roles` | `roles` |
| `Capabilities` | `capabilities` |
| `Principal.Type` | `principal_type` |
| `SessionID` | `sid` |
| `Issuer` | 已验证的 `iss` |
| `AuthnMethod` | 固定为 `oidc_jwt` |

这些 Claim 名称可配置，使企业现有身份系统可以映射到 AI Cloud，而不让 Domain Code 绑定某一个身份供应商。

## JWKS 生命周期

Verifier 在构造阶段主动获取 JWKS，使错误配置或无法访问的身份基础设施能够尽早 Fail Fast。Signing Key 使用有限 TTL 缓存；如果 Token 使用未知 `kid`，或恰好发生正常 Key Rotation，Verifier 会强制刷新 JWKS 一次并重新进行一次签名校验。

这个模型允许安全的正常密钥轮换，但不会接受 Token 自己提供的任意公钥。

## 失败模型

认证失败由 R7A API Boundary 转换成稳定的 `UNAUTHENTICATED` ErrorEnvelope。OIDC Metadata/JWKS 配置错误属于启动或配置失败，而不是每次请求的授权判断。

## 测试

R7B 核心测试使用进程内 TLS OIDC/JWKS Server 和真实 2048-bit RSA Signature，覆盖：

- Verified Claim 正确映射；
- Audience 不匹配拒绝；
- Expired Token 拒绝；
- 外部 System Principal 拒绝；
- JWKS Key Rotation 刷新；
- 缺失或重复 Authorization Header 拒绝。

## R7B 剩余工作

在 R7B 可以正式标记完成之前，还需要：

1. 把 Verifier Selection 接入 `cmd/api-server` 配置；
2. Trusted Ingress 仅保留为显式兼容模式；
3. 保证 Request/Trace Metadata 位于 Authentication 外层，使 401 也具备关联 ID；
4. 增加配置与 Server Wiring Test；
5. 运行完整 CI，并完成 Authentication Boundary Review。

RBAC/ABAC 属于 R7C；OpenAPI/Domain Convergence 属于 R7D。
