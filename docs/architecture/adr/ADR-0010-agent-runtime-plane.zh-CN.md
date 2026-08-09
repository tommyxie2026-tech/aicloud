# ADR-0010 Agent Runtime Plane

## 背景
AI Agent 需要独立执行环境。

## 架构
包括：
- Agent Runtime
- Tool Gateway
- Memory
- Workflow Engine
- Sandbox

## 原则
模型负责推理，策略负责决策，控制器负责执行。

## 目标
构建安全可控的 Agent 执行平面。