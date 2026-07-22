# AI Cloud API Specification v1

## API Principles

REST API + Async Task API + Streaming Events.

## Model API

```
GET /api/v1/models
POST /api/v1/models
```

Manage model registry.

## Agent API

```
GET /api/v1/agents
POST /api/v1/agents
```

Create and manage agents.

## Task API

```
POST /api/v1/tasks
GET /api/v1/tasks/{id}
```

Submit and query agent tasks.

## Workflow API

```
GET /api/v1/workflows
```

Manage workflow definitions.

## Streaming API

```
GET /api/v1/tasks/{id}/events
```

Provide execution events.

## Security

All APIs require identity, tenant context and policy evaluation.
