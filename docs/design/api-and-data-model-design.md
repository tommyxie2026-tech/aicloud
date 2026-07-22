# AI Cloud API and Data Model Design

## API Resources

## Model API

```
GET /api/v1/models
POST /api/v1/models
GET /api/v1/models/{id}
```

## Agent API

```
GET /api/v1/agents
POST /api/v1/agents
```

## Task API

```
POST /api/v1/tasks
GET /api/v1/tasks/{id}
POST /api/v1/tasks/{id}/cancel
```

## Workflow API

```
GET /api/v1/workflows
POST /api/v1/workflows
```

## Execution Event API

Async events:

- task.created
- agent.started
- tool.called
- sandbox.created
- execution.completed

# Data Model

## Model

```
id
name
provider
capabilities
pricing
license
risk_level
```

## Agent

```
id
name
model
workflow
tools
sandbox_policy
```

## Task

```
id
agent_id
input
status
result
cost
trace_id
```

## Tool

```
id
name
permission
risk_level
credential_policy
```

## Policy

```
id
subject
resource
action
decision
```

# Design Principle

The API is resource oriented and inspired by Kubernetes declarative design.
