# AI Cloud Developer Research Docs

**English** | [简体中文](README.zh-CN.md)

`devdocs/` stores implementation-oriented research notes that support AI Cloud architecture and engineering decisions.

These documents are different from product requirements and stable architecture specifications under `docs/`:

- `docs/` defines what AI Cloud intends to build and the contracts the project commits to;
- `devdocs/` studies external technologies, models, protocols, runtimes, and engineering patterns;
- conclusions in `devdocs/` are research inputs and do not become product commitments until promoted into an ADR, design document, API specification, or roadmap item.

## Bilingual documentation convention

AI Cloud `devdocs/` maintains English and Simplified Chinese versions.

```text
English compatibility entry: <name>.md
Chinese companion:          <name>.zh-CN.md
```

Maintenance requirements:

1. English and Chinese files must use matching section numbers and logical structure.
2. Tables, code blocks, Mermaid diagrams, configuration examples, and identifiers must remain aligned.
3. Facts, numbers, license conditions, limitations, and references must not change through translation.
4. Each document should provide a language switch near the title.
5. Architecture changes must update both languages in the same pull request.
6. When synchronization is temporarily impossible, mark `translation-status: pending` explicitly.
7. Keep model names, API fields, environment variables, file names, and code identifiers untranslated.
8. A translation may use natural language rather than literal word order, but it must preserve the same engineering conclusion and evidence boundary.

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
├── README.md
├── README.zh-CN.md
├── ai-execution-control-plane.md
├── ai-execution-control-plane.zh-CN.md
├── task-execution-contract.md
├── task-execution-contract.zh-CN.md
├── us-china-ai-industrial-stack-and-execution-os.md
├── us-china-ai-industrial-stack-and-execution-os.zh-CN.md
└── models/
    └── kimi-k3/
```

## Current studies

- [AI Execution Control Plane: Evolution from Model Platform to Task Execution Control Plane](ai-execution-control-plane.md)
- [AI Execution Control Plane：从模型平台到任务执行控制面的演进](ai-execution-control-plane.zh-CN.md)
- [Task Execution Contract](task-execution-contract.md)
- [Task Execution Contract：任务执行契约](task-execution-contract.zh-CN.md)
- [U.S.-China AI Industrial Stacks and AI Execution OS](us-china-ai-industrial-stack-and-execution-os.md)
- [中美 AI 产业栈与 AI Execution OS：从 2026 年川习会企业阵容看产业控制点](us-china-ai-industrial-stack-and-execution-os.zh-CN.md)
- [Kimi K3 Technical Architecture Study](models/kimi-k3/README.md)
- [Kimi K3 技术架构研究 — 简体中文](models/kimi-k3/README.zh-CN.md)
