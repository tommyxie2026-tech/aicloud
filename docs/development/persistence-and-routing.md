# Persistence, Operational Registry, and Routing

## Scope

This document describes the first production-oriented persistence and routing baseline.

Implemented capabilities:

- PostgreSQL-backed Model, Task, RouteDecision, and CostEvent repositories;
- embedded, transactional schema migrations;
- operational Model Registry fields;
- governed deterministic, efficient, specialist, and flagship routing classes;
- joint model, inference-effort, and service-tier selection records;
- immutable model-call CostEvent records;
- OpenAI-compatible commercial, private, or self-hosted provider adapter;
- authenticated provider health probing;
- provider registration without implicit production approval.

## Local PostgreSQL mode

Start dependencies:

```bash
make compose-up
```

Run migrations:

```bash
export AICLOUD_DATABASE_URL='postgres://aicloud:aicloud@localhost:5432/aicloud?sslmode=disable'
make migrate
```

Start the API with PostgreSQL persistence:

```bash
export AICLOUD_REPOSITORY_MODE=postgres
export AICLOUD_RUN_MIGRATIONS=true
make run
```

The full Compose application path is:

```bash
make compose-app
```

## OpenAI-compatible provider

The provider adapter supports commercial APIs and private or self-hosted endpoints that expose an OpenAI-compatible API.

Example configuration:

```bash
export AICLOUD_PROVIDER_ENABLED=true
export AICLOUD_PROVIDER_NAME=openai-compatible
export AICLOUD_PROVIDER_ENDPOINT=https://api.example.com/v1
export AICLOUD_PROVIDER_SECRET_ENV=OPENAI_API_KEY
export AICLOUD_PROVIDER_MODEL=model-name
export AICLOUD_PROVIDER_MODEL_VERSION=v1
export AICLOUD_PROVIDER_INPUT_PER_MILLION=1.0
export AICLOUD_PROVIDER_OUTPUT_PER_MILLION=5.0
export OPENAI_API_KEY='...'
```

A configured provider is not automatically eligible for production routing. Set `AICLOUD_PROVIDER_APPROVED=true` only after license, provenance, security, policy, and ownership evidence has been reviewed.

## Model lifecycle and admission

A model must satisfy all of the following before routing:

- lifecycle is `active`, or explicitly allowed `degraded`;
- immutable model version is `approved`;
- endpoint is not unhealthy;
- required capabilities are present;
- requested inference effort and service tier are supported;
- health and capacity evidence is fresh when required;
- quota and capacity are not explicitly exhausted;
- data-residency and budget constraints pass.

Negative quota or capacity values mean that a provider does not expose the signal. Zero means explicitly exhausted.

## Routing API

Create a task:

```http
POST /api/v1/tasks
```

Create a route decision:

```http
POST /api/v1/tasks/{taskId}/route
```

Example body:

```json
{
  "routeClass": "specialist",
  "requiredCapabilities": ["coding"],
  "inferenceEffort": "medium",
  "serviceTier": "standard",
  "estimatedInputTokens": 20000,
  "estimatedOutputTokens": 5000,
  "budget": 0.50,
  "currency": "USD",
  "evidenceVersion": "developer-agent-eval-v1",
  "policyVersion": "routing-policy-v1",
  "requireFreshSignals": true,
  "signalMaxAgeSeconds": 300
}
```

Retrieve decisions and costs:

```http
GET /api/v1/tasks/{taskId}/routes
GET /api/v1/tasks/{taskId}/costs
```

## Cost ledger

CostEvent records are append-only. Model usage creates separate input, output, and optional service-tier events. Retry attempts retain their attempt number and are not hidden from total task cost.

The current ledger refuses to aggregate mixed currencies without an explicit conversion process.

## Remaining work

This baseline does not yet execute a routed Task through Temporal. The next implementation stages are:

1. provider invocation activity and task-cost reconciliation;
2. Tool Gateway, OPA policy, short-lived credentials, and Kubernetes Sandbox;
3. Agent attack and privilege-boundary tests;
4. OpenTelemetry and reproducible evaluation evidence;
5. circuit breaker, live capacity ingestion, fallback execution, and admission control;
6. full license and provenance evidence workflow.
