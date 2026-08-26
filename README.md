# ChangeBlast

A local-first CLI that answers: *if I change this file, what am I likely
to affect?*

ChangeBlast analyzes a Git repository's dependency graph (and, in later
versions, its Git history and CI configuration) to estimate the blast
radius of a code change — deterministically, offline, with no account and
no cloud backend.

## Why

Understanding downstream impact of a change usually means either tribal
knowledge or grepping for imports by hand. ChangeBlast automates the
mechanical part of that question using evidence already in the
repository — imports, exports, history, CI — instead of guesswork or an
LLM.

## Status

**v0.1 — early vertical slice.** Currently implemented:

- `blast inspect <path>` — direct and indirect dependents of a JS/TS file
- `blast <path>` — convenience alias for `blast inspect <path>`
- `blast doctor` — environment/repository checks
- `blast version`

Not yet implemented: `diff`, `graph`, `history`, Git history signals, CI
analysis, risk scoring, `.changeblast.yml` configuration. See
[docs/architecture.md](docs/architecture.md) for what's scaffolded versus
what has real logic, and the roadmap below.

## Installation

```bash
git clone <repo-url>
cd changeblast
go build -o blast .
```

Requires Go 1.22+.

## Quick start

```bash
blast inspect src/auth/token.ts
```

```
Target
  src/auth/token.ts

Direct impact
  src/auth/middleware.ts

Indirect impact
  src/api/client.ts
```

Machine-readable output:

```bash
blast inspect src/auth/token.ts --json
```

See [docs/usage.md](docs/usage.md) for the full command reference,
resolution scope, and known limitations.

## Commands

| Command | Description |
|---|---|
| `blast inspect <path>` | Analyze direct/indirect dependents of a file |
| `blast <path>` | Alias for `blast inspect <path>` |
| `blast doctor` | Check environment and repository compatibility |
| `blast version` | Print version |

## Architecture

```
CLI -> Target Resolver -> Repository Scanner -> Language Analyzer
    -> Dependency Graph -> Impact Engine -> Risk Engine -> Output
```

Full write-up, including why the JS/TS analyzer is regex-based rather
than a full parser and the exact module-resolution scope: see
[docs/architecture.md](docs/architecture.md).

## Development

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing
```

Fixture repositories used by tests live under `testdata/fixtures/`.

## Roadmap

- Git analyzer: churn, co-change frequency, history window
- CI analyzer: GitHub Actions workflow relevance
- Risk engine: explainable, deterministic scoring with critical-path
  weighting
- `blast diff [<ref>]`, `blast graph <path>`, `blast history <path>`
- `.changeblast.yml` configuration (`criticalPaths`, `historyWindow` overrides)
- Additional language analyzers (Go, Python, Java, Rust)
- Additional CI providers (GitLab CI, Azure DevOps, Jenkins)
- Optional AI explanation layer (`blast diff --explain`) over the
  deterministic findings — never a replacement for them

## License

MIT — see [LICENSE](LICENSE).
