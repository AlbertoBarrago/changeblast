# Serval

<div align="center">
  <img src="docs/logo.svg" alt="Serval logo" width="180">
</div>

[![CI](https://github.com/AlbertoBarrago/serval/actions/workflows/ci.yml/badge.svg)](https://github.com/AlbertoBarrago/serval/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/AlbertoBarrago/serval)](https://github.com/AlbertoBarrago/serval/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/AlbertoBarrago/serval.svg)](https://pkg.go.dev/github.com/AlbertoBarrago/serval)
[![license](https://img.shields.io/github/license/AlbertoBarrago/serval)](LICENSE)

A local-first CLI that answers: *if I change this file, what am I likely
to affect?*

<div align="center">
  <img src="docs/serval.jpg" alt="A serval stalking through tall grass, its oversized ears raised" width="560">
  <p><sub>The namesake: a serval is a keen-eared hunter that finds what's
  hidden — like Serval finds a change's hidden downstream impact.
  Photo by <a href="https://www.wikidata.org/wiki/Q28147777">Diego Delso</a>,
  <a href="https://creativecommons.org/licenses/by-sa/4.0/">CC BY-SA 4.0</a>,
  via Wikimedia Commons (Tarangire National Park, Tanzania).</sub></p>
</div>

Serval analyzes a Git repository's dependency graph, Git history,
and CI configuration to estimate the blast radius of a code change,
with a deterministic, explainable risk score, offline, with no account
and no cloud backend.

> Renamed a few times while still young: ChangeBlast → Blast →
> Impactline → Serval. Same project, same history, new name (the
> "Blast" name collided with an unrelated, long-established Homebrew
> formula; "Impactline" was clean but long to type — see the
> changelog).

## Why

Understanding downstream impact of a change usually means either tribal
knowledge or grepping for imports by hand. Serval automates the
mechanical part of that question using evidence already in the
repository (imports, exports, history, CI) instead of guesswork or an
LLM.

## Status

**v0.1.** Implemented:

- `serval inspect [path]`: direct/indirect dependents, Git history,
  relevant CI workflows, and an explainable risk score for a file.
  Defaults to `.` (the current directory) when no path is given; for a
  directory, every module inside it is analyzed and reported as a
  risk-sorted summary.
- `serval diff [<ref>]`: the same analysis for every file changed
  against `<ref>` (default `HEAD`, i.e. uncommitted changes)
- `serval graph <path>`: one-level dependency/dependent graph for a
  file
- `serval history [path]`: Git churn and co-change frequency,
  defaults to `.` when no path is given
- `serval <path>`: convenience alias for `serval inspect <path>`
- `serval doctor`: environment/repository checks, including whether
  a local Ollama daemon is reachable
- `serval version`, shell completion
  (`serval completion bash|zsh|fish`), generated man pages
  (`man serval`)
- `--json` on every analysis command; `--output <file>`/`-o` to write a
  report to disk instead of stdout; `--fail-on <level>` (exit code 2)
  on `inspect`/`diff` for CI gating
- `--explain` (on `inspect` and `diff`): asks an AI provider to explain
  each result's risk score in natural language. Off by default, makes
  no network call or subprocess spawn unless passed explicitly, and can
  never alter the deterministic score, only explain it.
  `--explain-provider` picks the backend: `ollama` (default, a local
  daemon) or `claude`/`codex`/`gemini` (a local CLI you already have
  installed and signed in — reuses your existing subscription, no API
  key needed). On a directory or multi-file diff, this is one call per
  file, sequentially — can be slow.
- TTY-aware colored output, respecting `NO_COLOR`
- Language support: JavaScript/TypeScript (relative ESM/CommonJS
  imports, `tsconfig.json` `paths`/`baseUrl`), Go (imports resolved
  against `go.mod`, standard library and external modules recorded as
  external), Python (plain and from-imports, including relative
  imports; repository root treated as the sole `sys.path` entry,
  standard library/third-party imports recorded as external), Java
  (plain, type-wildcard, and static imports; each file's own `package`
  declaration derives its source root, standard/Maven-Gradle
  dependency imports recorded as external), and C (quoted `#include`
  only, resolved relative to the including file; `.c`/`.h`, not C++).

- `.serval.yml`: optional per-repository overrides for the
  critical-path keyword list and the Git history window (see
  [docs/usage.md](docs/usage.md)).
- CI relevance: GitHub Actions (`.github/workflows/*.yml`), GitLab CI
  (`.gitlab-ci.yml`), Azure DevOps Pipelines (`azure-pipelines.yml`),
  and Jenkins declarative pipelines (`Jenkinsfile`). This completes the
  originally planned v0.1 CI provider set.

See [docs/architecture.md](docs/architecture.md) for what's scaffolded
versus what has real logic, and the roadmap below.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install AlbertoBarrago/tap/serval
```

Published automatically on tagged releases via GoReleaser: see
[.goreleaser.yml](.goreleaser.yml) and
[AlbertoBarrago/homebrew-tap](https://github.com/AlbertoBarrago/homebrew-tap).

### From source

```bash
git clone https://github.com/AlbertoBarrago/serval
cd serval
go build -o serval .
```

Requires Go 1.22+.

### Shell completion

```bash
# zsh
echo 'source <(serval completion zsh)' >> ~/.zshrc

# bash
echo 'source <(serval completion bash)' >> ~/.bashrc

# fish
serval completion fish > ~/.config/fish/completions/serval.fish
```

## Quick start

```bash
serval inspect src/auth/token.ts
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
  HIGH: 82/100
  +28  14 downstream modules
  +20  critical path (matched "auth" in src/auth/token.ts)
  +14  high historical churn (7 changes)
  +12  3 frequently co-changed modules
  +8   1 CI workflow(s) affected
```

Machine-readable output:

```bash
serval inspect src/auth/token.ts --json
```

CI gating:

```bash
serval diff --fail-on high   # exits 2 if any changed file scores HIGH
```

AI explanation (requires `ollama serve` running locally):

```bash
serval inspect src/auth/token.ts --explain
```

See [docs/usage.md](docs/usage.md) for the full command reference,
resolution scope, and known limitations.

## Commands

| Command | Description |
|---|---|
| `serval inspect [path]` | Full analysis (impact, history, CI, risk) for a file, or a risk-sorted summary for a directory. Defaults to `.` |
| `serval diff [<ref>]` | Full analysis for every file changed against `<ref>` |
| `serval graph <path>` | One-level dependency/dependent graph for a file |
| `serval history [path]` | Git churn and co-change frequency. Defaults to `.` |
| `serval <path>` | Alias for `serval inspect <path>` |
| `serval doctor` | Check environment and repository compatibility |
| `serval version` | Print version |

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

- A raw OpenAI/Anthropic/Gemini API integration (bring-your-own-key) as
  an alternative to the CLI-based `claude`/`codex`/`gemini`
  `--explain-provider` choices already available

## License

MIT, see [LICENSE](LICENSE).
