# On-call rotation scheduler

This pure Go command-line program builds an on-call schedule for a date range,
including holidays, per-engineer blackout dates, rotation start position, and a
fairness summary.

## Commands

```sh
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...
GOTOOLCHAIN=local go run . --smoke-test
GOTOOLCHAIN=local go run . --roster alice,bob --start 2026-03-02 --end 2026-03-04
```

The project has no external services or runtime dependencies.
