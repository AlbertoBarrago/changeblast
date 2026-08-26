# Changelog

All notable changes to ChangeBlast are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] — 2026-08-26

Initial public release.

### Added

- `blast inspect <path>` — direct/indirect JS/TS dependency impact, Git
  history signals, relevant GitHub Actions workflows, and a deterministic,
  explainable risk score.
- `blast diff [<ref>]` — the same analysis for every JS/TS file changed
  against `<ref>` (default `HEAD`), including uncommitted and untracked
  changes.
- `blast graph <path>` — one-level dependency/dependent graph for a file.
- `blast history <path>` — Git churn and co-change frequency, bounded to
  the last 90 days or 200 commits touching the file.
- `blast doctor` — environment and repository compatibility checks.
- `blast <path>` convenience alias for `blast inspect <path>`.
- `--json` on every analysis command; `--fail-on <level>` on
  `inspect`/`diff` mapped to exit code 2 for CI gating.
- TTY-aware, `NO_COLOR`-respecting colored output.
- Shell completion (`blast completion bash|zsh|fish`) and a generated
  man page (`man blast`).
- JS/TS module resolution: relative ESM/CommonJS imports, `tsconfig.json`
  `baseUrl`/`paths` aliases, extension and index resolution.
- Homebrew distribution via `AlbertoBarrago/homebrew-tap`.

### Known limitations

See [docs/usage.md](docs/usage.md#known-limitations) and
[docs/architecture.md](docs/architecture.md) — notably: no
`node_modules` resolution, no dynamic `import()` traversal, single-package
repositories only (no monorepo workspace awareness), GitHub Actions-only
CI analysis, and no `.changeblast.yml` configuration yet.
