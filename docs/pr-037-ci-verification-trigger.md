# PR-037 CI Verification Trigger

This draft pull request exists only to trigger the `pull_request` path of `.github/workflows/go-test.yml`.

Verification target:

```text
- go mod tidy leaves module metadata unchanged
- go test ./... succeeds
```

This file has no runtime effect and should not be merged unless it remains useful as CI documentation.
