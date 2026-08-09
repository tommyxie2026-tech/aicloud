# AI Cloud S0 契约冻结

## 目的

在继续编码之前冻结关键契约，避免架构漂移和后期重构。

## 核心原则

1. Task 是业务执行聚合根。
2. Tenant 和 Project 是安全边界。
3. Workflow 负责编排，不拥有业务事实。
4. Provider 是 Adapter，不是 Model Catalog。
5. Router 只在允许候选中选择。
6. Policy 决定是否允许，Router 决定如何优化。
7. Tool Gateway 是唯一副作用入口。
8. 所有副作用必须产生 Evidence。
9. 所有成本必须归属于 Task。
10. 所有生产模型必须具备 Admission 和 Evaluation 证据。

## 运行时责任边界

PostgreSQL：
- 业务状态；
- 查询状态；
- 资源所有权。

Workflow Engine：
- 执行编排；
- Retry 和 Recovery。

Task Events：
- 不可变业务历史。

Observability：
- 运行重建。

## 非目标

v0.1 不做：

- 过早微服务拆分；
- 无限制自治 Agent；
- Provider 专属业务逻辑；
- Agent 直接访问企业资源。

## Review Gate

任何 Slice 开始编码前必须完成：

- Domain Contract Review；
- API Contract Review；
- Security Boundary Review；
- Persistence Contract Review；
- Test Acceptance 定义；
- 中英文文档同步。
