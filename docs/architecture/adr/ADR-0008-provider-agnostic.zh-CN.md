# ADR-0008 Provider Agnostic Architecture

## 背景
AI Cloud 不应绑定单一模型供应商。商业模型、开放模型和私有部署模型需要统一管理。

## 决策
引入 Provider Abstraction，使应用依赖 AI Cloud API，而不是直接依赖 OpenAI、Gemini、Claude 等接口。

## 核心原则
- Provider 可插拔
- Model Registry 独立
- Router 支持迁移
- 支持商业 API 与自托管模型

## 目标
实现模型供应商无关的 AI 基础设施。