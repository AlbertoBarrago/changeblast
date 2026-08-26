# Changelog

All notable changes to ChangeBlast are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.4] (2026-08-26)

### Added

- `blast inspect <file> --explain`: asks a local Ollama model to explain
  the risk score in natural language, using only the already-computed
  deterministic findings as input. Off by default, no network call
  unless passed. New flags `--explain-model` and `--explain-host`.
  Single-file targets only in this release.
- `blast doctor` now reports whether a local Ollama daemon is reachable
  (informational only; the one network call `doctor` makes, always to
  localhost/`$OLLAMA_HOST`).

### Fixed

- `blast inspect` and `blast history` required a path argument and
  failed with a bare `accepts 1 arg(s), received 0` when run without
  one. Both now default to `.` (the current directory) when omitted,
  consistent with `blast inspect .` already being supported.

## [0.1.3] (2026-08-26)

### Fixed

- Generated man pages (`docs/*.1`) were never packaged into release
  archives or installed by the Homebrew formula, so `man blast` printed
  nothing after installation. The archive now bundles them and the
  formula installs them into `man1`.

## [0.1.2] (2026-08-26)

### Added

- `blast inspect <directory>` (including `blast inspect .`): analyzes
  every module inside the directory and renders a risk-sorted summary
  instead of one full report per file.
- `--output <file>`/`-o` on `inspect`, `diff`, `graph`, and `history`:
  write the report to disk instead of stdout.
- Go language support: imports resolved against `go.mod` (single-line
  and grouped forms, aliased/blank/dot), one edge per file in the
  target package; standard library and external modules recorded as
  external.

### Fixed

- The `blast <path>` alias now accepts `--json` and `--fail-on`, which
  previously only worked on `blast inspect <path>` directly.

## [0.1.1] (2026-08-26)

### Added

- TTY-aware, `NO_COLOR`-respecting colored output for risk levels and
  `blast doctor` status markers.
- `blast version` now reports the real commit, build date, Go version,
  and platform.

### Fixed

- `blast diff` rescanned the whole repository once per changed file; it
  now scans once and reuses the graph.

### Changed

- Internal cleanup: removed dead code, deduplicated the "frequently
  co-changed" counting logic (previously reimplemented three times),
  and expanded unit test coverage across `internal/graph`,
  `internal/impact`, `internal/output`, and tsconfig resolution in
  `internal/repository`.

## [0.1.0] (2026-08-26)

Initial public release.

### Added

- `blast inspect <path>`: direct/indirect JS/TS dependency impact, Git
  history signals, relevant GitHub Actions workflows, and a deterministic,
  explainable risk score.
- `blast diff [<ref>]`: the same analysis for every JS/TS file changed
  against `<ref>` (default `HEAD`), including uncommitted and untracked
  changes.
- `blast graph <path>`: one-level dependency/dependent graph for a file.
- `blast history <path>`: Git churn and co-change frequency, bounded to
  the last 90 days or 200 commits touching the file.
- `blast doctor`: environment and repository compatibility checks.
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
[docs/architecture.md](docs/architecture.md), notably: no
`node_modules` resolution, no dynamic `import()` traversal, single-package
repositories only (no monorepo workspace awareness), GitHub Actions-only
CI analysis, and no `.changeblast.yml` configuration yet.
