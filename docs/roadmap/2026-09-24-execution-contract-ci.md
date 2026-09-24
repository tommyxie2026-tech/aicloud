# aicloud ↔ computecloud CI 联调

## 目标
将 Execution Contract 从文档约定升级为持续集成门禁。

## 当前链路
ExecutionRequest → ComputecloudJobMapper → computecloud v0.2 JobSpec → ComputecloudProvider → /v1/jobs。

## CI
GitHub Actions: `.github/workflows/execution-contract-ci.yml`

CI 验证：
- Provider/Outcome 状态边界；
- HTTP Submit/Get adapter；
- ExecutionRequest → JobSpec mapper；
- gofmt；
- execution package regression。

## 下一阶段
- events cursor 替代轮询；
- ArtifactRef；
- TraceID；
- 使用真实 computecloud server + mock Worker 的跨仓库 E2E。

说明：当前 CI 是 aicloud 内的 contract/adapter CI，还不是启动两个真实服务的跨仓库系统测试。
