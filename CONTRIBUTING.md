# Contributing to ChangeBlast

## Development setup

Requires Go 1.22+.

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l .   # must print nothing before committing
```

## Principles

- CLI-first, local-first, offline-capable, deterministic by default.
- No SaaS, web UI, backend, auth system, database, or telemetry. See the
  Product Principles section of the project brief for the full list.
- Prefer the Go standard library. New dependencies need explicit
  justification.
- Keep language analyzers, CI providers, and the core pipeline decoupled:
  `internal/analyzer/*` and `internal/ci/*` implement small interfaces
  consumed by the scanner; they must not be imported directly by `cmd/`
  or `internal/risk`.
- Every documented limitation (module resolution scope, critical-path
  keyword list, history window) must stay documented in
  `docs/architecture.md` when it changes — do not silently narrow or
  widen behavior.

## Commit style

Conventional Commits (`feat:`, `fix:`, `refactor:`, `chore:`, `docs:`,
`test:`), one logical change per commit.

## Adding a language analyzer

Implement `analyzer.Analyzer` (`internal/analyzer/analyzer.go`) in a new
`internal/analyzer/<language>` package, and register it in
`repository.NewScanner`. Do not add language-specific branching to
`internal/repository/scanner.go` beyond analyzer selection.

## Adding a CI provider

Same pattern: implement the CI analyzer contract in
`internal/ci/<provider>`, keep provider-specific parsing out of the core
pipeline.

## Tests

- Unit tests alongside the code they test (`_test.go`).
- Fixture repositories for scanner/resolver behavior go under
  `testdata/fixtures/<name>/`; prefer a realistic small fixture over
  mocking the filesystem.
- Do not depend on network access or external services in tests.
