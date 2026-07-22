# AI Cloud Kubernetes Architecture

## Deployment Model

AI Cloud runs as cloud native services.

```
Kubernetes Cluster

Control Plane
  - API Server
  - Model Service
  - Policy Service

Execution Plane
  - Agent Runtime
  - Workflow Worker
  - Sandbox Worker

Data Services
  - PostgreSQL
  - Redis
  - Object Storage
```

## Future Extensions

- AI Cloud Operator
- CRD based resource management
- Helm deployment
- Multi cluster execution
