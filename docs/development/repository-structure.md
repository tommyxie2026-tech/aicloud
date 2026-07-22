# AI Cloud Repository Structure

Recommended layout:

```
aicloud/

cmd/
  api-server/
  worker/

services/
  control-plane/
  model-service/
  agent-runtime/
  workflow-service/
  tool-gateway/
  policy-service/

pkg/
  protocol/
  auth/
  telemetry/

api/
  openapi/

deploy/
  helm/
  docker/

tests/

docs/
```

## Engineering Principle

Keep domain boundaries clear before introducing microservices.

Start with modular architecture and evolve based on operational needs.
