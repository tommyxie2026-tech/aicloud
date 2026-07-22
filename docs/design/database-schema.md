# AI Cloud Database Schema Design

## Purpose

Define persistent data model for AI Cloud v0.1.

## Core Entities

### tenant

Enterprise isolation boundary.

Fields:
- id
- name
- status
- created_at

### user

Identity and access.

Fields:
- id
- tenant_id
- role
- status

### model

AI model registry.

Fields:
- id
- provider
- name
- version
- capability
- pricing
- license

### agent

Agent definition.

Fields:
- id
- tenant_id
- model_id
- workflow_id
- policy_id

### task

Agent execution request.

Fields:
- id
- agent_id
- input
- status
- result
- cost
- trace_id

### workflow_execution

Long-running execution state.

Fields:
- id
- task_id
- state
- checkpoint

### tool

External capability registry.

Fields:
- id
- name
- permission
- risk_level

### audit_event

Security and compliance records.

Fields:
- id
- actor
- action
- resource
- timestamp

## Design Principle

The schema follows Kubernetes-style declarative resource management.
