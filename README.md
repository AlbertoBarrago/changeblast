# ChangeBlast

[![CI](https://github.com/AlbertoBarrago/changeblast/actions/workflows/ci.yml/badge.svg)](https://github.com/AlbertoBarrago/changeblast/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/AlbertoBarrago/changeblast)](https://github.com/AlbertoBarrago/changeblast/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/AlbertoBarrago/changeblast.svg)](https://pkg.go.dev/github.com/AlbertoBarrago/changeblast)
[![license](https://img.shields.io/github/license/AlbertoBarrago/changeblast)](LICENSE)

A local-first CLI that answers: *if I change this file, what am I likely
to affect?*

ChangeBlast analyzes a Git repository's dependency graph, Git history,
and CI configuration to estimate the blast radius of a code change — with
a deterministic, explainable risk score, offline, with no account and no
cloud backend.

## Why

Understanding downstream impact of a change usually means either tribal
knowledge or grepping for imports by hand. ChangeBlast automates the
mechanical part of that question using evidence already in the
repository — imports, exports, history, CI — instead of guesswork or an
LLM.

## Status

**v0.1.** Implemented:

- `blast inspect <path>` — direct/indirect dependents, Git history,
  relevant CI workflows, and an explainable risk score for a JS/TS file
- `blast diff [<ref>]` — the same analysis for every JS/TS file changed
  against `<ref>` (default `HEAD`, i.e. uncommitted changes)
- `blast graph <path>` — one-level dependency/dependent graph for a file
- `blast history <path>` — Git churn and co-change frequency for a file
- `blast <path>` — convenience alias for `blast inspect <path>`
- `blast doctor` — environment/repository checks
- `blast version`, shell completion (`blast completion bash|zsh|fish`),
  generated man pages (`man blast`)
- `--json` on every analysis command; `--fail-on <level>` (exit code 2)
  on `inspect`/`diff` for CI gating

Not yet implemented: `.changeblast.yml` configuration, additional
language analyzers, additional CI providers, the optional AI explanation
layer. See [docs/architecture.md](docs/architecture.md) for what's
scaffolded versus what has real logic, and the roadmap below.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap AlbertoBarrago/tap
brew install changeblast
```

Published automatically on tagged releases via GoReleaser — see
[.goreleaser.yml](.goreleaser.yml) and
[AlbertoBarrago/homebrew-tap](https://github.com/AlbertoBarrago/homebrew-tap).

### From source

```bash
git clone https://github.com/AlbertoBarrago/changeblast
cd changeblast
go build -o blast .
```

Requires Go 1.22+.

### Shell completion

```bash
# zsh
echo 'source <(blast completion zsh)' >> ~/.zshrc

# bash
echo 'source <(blast completion bash)' >> ~/.bashrc

# fish
blast completion fish > ~/.config/fish/completions/blast.fish
```

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

CI
  integration-auth.yml

Git history
  7 significant changes (last 90 days)
  3 frequently co-changed modules

Risk
  HIGH — 82/100
  +28  14 downstream modules
  +20  critical path (matched "auth" in src/auth/token.ts)
  +14  high historical churn (7 changes)
  +12  3 frequently co-changed modules
  +8   1 CI workflow(s) affected
```

Machine-readable output:

```bash
blast inspect src/auth/token.ts --json
```

CI gating:

```bash
blast diff --fail-on high   # exits 2 if any changed file scores HIGH
```

See [docs/usage.md](docs/usage.md) for the full command reference,
resolution scope, and known limitations.

## Commands

| Command | Description |
|---|---|
| `blast inspect <path>` | Full analysis (impact, history, CI, risk) for a file |
| `blast diff [<ref>]` | Full analysis for every file changed against `<ref>` |
| `blast graph <path>` | One-level dependency/dependent graph for a file |
| `blast history <path>` | Git churn and co-change frequency for a file |
| `blast <path>` | Alias for `blast inspect <path>` |
| `blast doctor` | Check environment and repository compatibility |
| `blast version` | Print version |

## Architecture

```
CLI -> Target Resolver -> Repository Scanner -> Language Analyzer
    -> Dependency Graph -> Impact Engine -> Risk Engine -> Output
```

Full write-up, including why the JS/TS analyzer is regex-based rather
than a full parser, the exact module-resolution scope, and the risk
scoring model: see [docs/architecture.md](docs/architecture.md).

## Development

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing
make man     # regenerate docs/*.1 after changing command help text
```

Fixture repositories used by tests live under `testdata/fixtures/`.

## Roadmap

- `.changeblast.yml` configuration (`criticalPaths`, `historyWindow` overrides)
- Additional language analyzers (Go, Python, Java, Rust)
- Additional CI providers (GitLab CI, Azure DevOps, Jenkins)
- Optional AI explanation layer (`blast diff --explain`) over the
  deterministic findings — never a replacement for them
- First tagged release (`v0.1.0`) to publish the Homebrew formula and
  cross-platform binaries via GoReleaser

## License

MIT — see [LICENSE](LICENSE).
