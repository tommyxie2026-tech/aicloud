# AI Cloud Developer Research Docs

`devdocs/` stores implementation-oriented research notes that support AI Cloud architecture and engineering decisions.

These documents are different from product requirements and stable architecture specifications under `docs/`:

- `docs/` defines what AI Cloud intends to build and the contracts the project commits to;
- `devdocs/` studies external technologies, models, protocols, runtimes, and engineering patterns;
- conclusions in `devdocs/` are research inputs and do not become product commitments until promoted into an ADR, design document, API specification, or roadmap item.

## Research rules

Every study should distinguish four evidence levels:

1. **Published fact** — stated in an official paper, model card, repository, configuration, or license.
2. **Code-verifiable fact** — directly visible in released source or configuration.
3. **Engineering inference** — a reasoned conclusion derived from published facts.
4. **Unknown or unverified** — not released, ambiguous, or not independently reproduced.

Research documents should:

- pin the observation date;
- prefer primary sources;
- identify vendor-reported benchmark results;
- avoid treating open weights as equivalent to complete open source;
- record deployment, license, provenance, evaluation, and operational constraints;
- explain direct implications for AI Cloud without silently changing the product roadmap.

## Directory

```text
devdocs/
└── models/
    └── kimi-k3/
```

## Current studies

- [Kimi K3 technical architecture](models/kimi-k3/README.md)
