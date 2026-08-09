# Deployment and Testing Specification

## Environments
Required: local, CI, dev, staging, production. Local/CI may use reduced dependencies; staging must exercise the same security and persistence boundaries as production.

## Local development
Docker Compose provides PostgreSQL, Redis, Temporal development server, OPA, and optional S3-compatible object storage. API and worker may run on host for fast iteration. MockProvider is mandatory and deterministic so tests do not depend on external model APIs.

## Kubernetes production topology

```text
Ingress/API Gateway
  -> aicloud-api Deployment (>=2)
  -> aicloud-worker Deployment (>=2)

External/stateful dependencies:
  PostgreSQL HA
  Redis HA/managed
  Temporal cluster/managed
  OPA sidecar/service or embedded policy adapter
  Object Storage
  OpenTelemetry Collector

Execution:
  Sandbox Jobs in restricted namespace(s)
  self-hosted model endpoints in dedicated GPU namespace/cluster
```

API is stateless. Worker is horizontally scalable. Durable state lives in PostgreSQL/Temporal/object storage. Redis is not the source of truth.

## Namespace model
Initial production namespaces:

```text
aicloud-system      API/worker/control components
aicloud-sandbox     default sandbox jobs
aicloud-models      self-hosted model runtimes
aicloud-observe     optional telemetry collectors
```

Tenant identity is enforced logically and by workload labels/service accounts; do not create one Kubernetes namespace per tenant by default. High-isolation tenants may later receive dedicated namespace/cluster profiles.

## Kubernetes security baseline
API/worker: non-root, read-only root filesystem where practical, dropped capabilities, resource requests/limits, PodDisruptionBudget, readiness/liveness/startup probes, restricted service accounts, no wildcard cluster-admin.

Sandbox: stricter profile defined in runtime-security-flow. NetworkPolicy defaults deny; egress is explicit.

## Configuration
Configuration is environment-based with typed validation at startup. Secrets are referenced from a secret manager/Kubernetes Secret adapter and never committed. Provider credentials use named credential references in Registry; raw keys do not appear in ModelVersion records.

## Availability targets for initial production
API target: 99.9% monthly availability excluding upstream provider outage. No single API/worker pod failure may lose Task state. Provider outage must degrade through policy-compliant fallback or explicit unavailable status, not uncontrolled retry.

## Observability minimum
OpenTelemetry traces, Prometheus-compatible metrics, structured JSON logs. Required metrics: request rate/error/latency; task state counts/duration; provider call latency/error/rate-limit; routing candidate rejection reasons; tool/policy/approval counts; sandbox duration/failure; token/usage/cost; workflow retry; queue/backlog.

## CI gates
Every PR must run:

```text
gofmt check
go test ./...
go vet ./...
build API + worker
migration validation
OpenAPI validation
unit tests
repository integration tests with PostgreSQL
policy tests
multi-tenant isolation tests
provider contract tests using MockProvider
Helm lint/template
```

Security scanning and image/SBOM checks are added before production release.

## Test pyramid
Unit tests cover domain invariants and deterministic routing. Integration tests cover PostgreSQL, Redis where used, OPA, Temporal adapter, and Tool/Sandbox adapters with safe fakes. Contract tests ensure every Provider adapter passes the same behavior suite. End-to-end tests run the reference Task through API -> workflow -> router -> MockProvider -> policy -> fake tool/sandbox -> validation -> cost/audit/trace.

## Mandatory negative tests

- forged tenant_id cannot access another tenant;
- repository query without tenant scope is rejected by API/design tests and RLS where enabled;
- fallback never selects policy-ineligible model;
- policy engine outage blocks side effects;
- expired approval cannot execute;
- expired credential cannot execute;
- duplicate Tool invocation with same idempotency key does not duplicate side effect;
- unknown Tool risk class is denied;
- sandbox cannot reach disallowed network or host resources;
- provider timeout respects total Task deadline;
- cost events are not duplicated on workflow retry;
- trace/audit records contain IDs but no configured secret fields.

## Release gates
A model/provider version is promoted only after provider contract tests, routing compatibility, safety/policy checks, and evaluation threshold pass. An application release is promoted only after migrations, rollback/roll-forward plan, smoke test, tenant isolation suite, and reference scenario pass.

## Disaster and recovery expectations
PostgreSQL and Temporal data require backups and tested restore. Object artifacts follow retention policy. Redis loss may reduce performance but must not corrupt source-of-truth state. Workflow replay must preserve tenant context and idempotency.

## Definition of production-ready
Production-ready means at least two API replicas, durable workflow/persistence, tenant isolation tests, fail-closed policy boundary, bounded retry/fallback, complete audit/cost/trace for reference scenario, health/readiness probes, backup/restore procedure, and a tested upgrade path.