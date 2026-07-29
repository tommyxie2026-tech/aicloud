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
  architecture/
  api/
  design/
  deployment/
  development/
  operations/
  roadmap/

devdocs/
  models/
    kimi-k3/
```

## Documentation boundaries

### `docs/`

Stable product and engineering documents:

- architecture decisions;
- API and data contracts;
- implementation plans;
- deployment and operations requirements;
- accepted security and governance boundaries.

### `devdocs/`

Implementation-oriented external technology research:

- model architecture studies;
- inference-engine investigations;
- protocol experiments;
- deployment feasibility notes;
- license and supply-chain analysis;
- technical findings that may later be promoted into an ADR or design document.

A conclusion in `devdocs/` does not become a product commitment until it is reviewed and promoted into the stable `docs/` hierarchy.

## Engineering Principle

Keep domain boundaries clear before introducing microservices.

Start with modular architecture and evolve based on operational needs.
