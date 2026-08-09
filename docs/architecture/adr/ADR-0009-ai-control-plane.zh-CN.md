# ADR-0009 AI Cloud Control Plane

## 背景
随着模型数量增加，AI Cloud 需要从模型网关升级为控制平面。

## 架构

- Model Registry
- Evaluation
- Governance
- Policy Engine
- Intelligent Router

## 决策
模型选择由策略驱动，而不是业务代码决定。

## 目标
形成可迁移、可治理、可扩展的企业 AI 控制中心。