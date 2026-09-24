# aicloud ↔ computecloud 联调进度

## 2026-09-24

进入 E2/C1 代码联调阶段。

已完成：ExecutionProvider v1alpha1、computecloud external contract、ComputecloudProvider HTTP adapter；Submit/Get/Cancel 已映射到 computecloud v1 Jobs API；Watch 首版状态轮询；aicloud execution ID 用作 Idempotency-Key；继续保持 ExecutionCompleted 与 OutcomeVerified 分离。

下一步：正式 Job Mapper；Watch 切换 events cursor；ArtifactRef；TraceID；mock worker E2E；Verifier failure → replan/retry。

约束：adapter 不 import computecloud Go module，双方仅通过版本化 HTTP/JSON contract 集成。
