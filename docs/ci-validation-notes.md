# CI Validation Notes

## Goal

This document records how CI validates the repository during the stabilization phase.

## Current Workflow

Workflow file:

```text
.github/workflows/go-test.yml
```

Current validation sequence:

```text
1. checkout
2. setup Go 1.22
3. go mod tidy
4. git diff --exit-code -- go.mod go.sum
5. go test ./...
```

## Why go mod tidy Runs Before Tests

`gopkg.in/yaml.v3` was added for the `integrations/gitops/yamlio` package.

A real `go mod tidy` run is required to generate and verify `go.sum`.

The stabilization workflow should fail if:

```text
- go.sum is missing
- go.mod changes after tidy
- go.sum changes after tidy
- tests do not compile or pass
```

## Important Interpretation

A failure in the `Verify module files are tidy` step means dependency metadata is not committed or not reproducible.

A failure in the `Run tests` step means compile or test execution failed after module files are already tidy.

## Current Boundary

This CI does not perform:

```text
- integration tests against real OpenAI-compatible providers
- Kubernetes API server tests
- GitHub PR creation tests
- kubectl apply
- controller-runtime tests
```

## Required Local Commands

Run locally before pushing stabilization changes:

```bash
go mod tidy
go test ./...
```

Commit `go.sum` if `go mod tidy` generates it.

Do not hand-write `go.sum` hashes.
