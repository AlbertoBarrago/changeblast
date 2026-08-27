# Changelog

All notable changes to Serval are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.20] (2026-08-27)

### Fixed

- `matchPathPattern` panicked on a tsconfig `paths` pattern whose fixed
  parts were longer than the specifier (e.g. `"@x/*/y"` against
  `"@x/z"`): the wildcard slice went out of range. Such patterns now
  simply don't match.
- tsconfig `paths` resolution was nondeterministic with overlapping
  patterns (map iteration order); patterns are now matched most-specific
  first, consistently across runs. Substitution targets are also tried
  in declaration order with the first one existing on disk winning
  (TypeScript's behavior), instead of always using the first.
- `go.mod` parsing kept trailing line comments in the module path
  (`module example.com/m // note`), making every local Go import
  resolve as external.
- `GoResolver` included `_test.go` files as dependents of every imported
  package, inflating impact counts and risk scores; test files are now
  excluded.
- `repositoryRoot` silently fell back to the working directory outside
  any git repository, producing degraded git-less analyses; it now
  returns an explicit error. It also accepts a `.git` file (worktrees,
  submodules), previously treated as "not a repository".
- `--fail-on` was validated only after a full analysis run; an invalid
  value now fails fast. Values are case/space tolerant (`" High "`).
- Risk score breakdown now stays consistent with the clamped total
  above 100 instead of summing to the unclamped value.
- Critical-path keywords match whole path segments only: `auth` no
  longer matches `src/author/bio.ts` (documented behavior, previously
  contradicted by the code).
- Unreadable files no longer abort a whole repository scan; they are
  reported on stderr and skipped. Same for malformed CI workflow files,
  per-file analysis errors in `inspect`/`diff`, and unanalyzable modules
  in directory mode: every skip is now visible instead of silent.
- CI path-filter globs: `**` now spans whole path segments only
  (`src/**` no longer matches `srcfoo/x.ts`), matching GitHub Actions
  semantics; compiled globs are cached.
- Co-change analysis shells out to git once (a single batched
  `git show`) instead of once per commit (up to 200 subprocesses).

### Changed

- `make lint` now fails on unformatted files (`gofmt -l` output no
  longer ignored).

### Removed

- Dead code: unused `directSet` in impact.Compute, hand-rolled
  `trimPrefix` in doctor.

### Internal

- One shared, parametrized comment-stripping lexer
  (`internal/strip.Comments`) replaces six near-identical per-language
  copies.
- Shared helpers for manifest lookup (`findUpward`), root-relative path
  rendering (`output.relPath`), multi-result JSON encoding, and the
  `--json/--fail-on/--output` flag trio (`analysisFlags`).
- New tests: `--fail-on` handling, `repositoryRoot` (including `.git`
  files), glob matching, tsconfig panic/determinism/fallback, the shared
  comment lexer, and a diff-from-subdirectory smoke test.

## [0.1.18] (2026-08-27)

### Changed

- **Project renamed from Impactline to Serval** (third rename in one
  day). "Impactline" was collision-free but long to type on every
  invocation; picked a short, evocative name instead — a serval is a
  keen-eared hunter that finds what's hidden, a fitting stand-in for
  finding a change's hidden downstream impact — verified clean on
  Homebrew, npm, PyPI, and GitHub before committing to it. Breaking
  changes for existing users, same shape as the previous renames:
  - GitHub repository: `AlbertoBarrago/impactline` →
    `AlbertoBarrago/serval` (GitHub redirects the old URL automatically)
  - Go module path: `github.com/AlbertoBarrago/impactline` →
    `github.com/AlbertoBarrago/serval`
  - Binary/command name: `impactline` → `serval`
  - Config file: `.impactline.yml` → `.serval.yml`, no
    backward-compat fallback
  - Man pages: `impactline.1`/`impactline-*.1` →
    `serval.1`/`serval-*.1`
  - Homebrew: `brew install AlbertoBarrago/tap/serval` replaces
    `brew install AlbertoBarrago/tap/impactline`; the `impactline`
    formula (live for a few hours) is kept in the tap, deprecated,
    pointing at the new one — same treatment `blast` and `changeblast`
    got in v0.1.15–v0.1.17

## [0.1.17] (2026-08-27)

### Changed

- **Project renamed from Blast to Impactline** (second rename in one
  day — see v0.1.15/v0.1.16). "Blast" turned out to collide with
  NCBI's long-established BLAST bioinformatics formula on Homebrew
  (`brew install blast` silently installed the wrong ~150MB package);
  rather than keep working around that with fully-qualified install
  instructions forever, picked a name verified against Homebrew, npm,
  PyPI, and GitHub repositories before committing to it this time.
  Breaking changes for existing users, same shape as the previous
  rename:
  - GitHub repository: `AlbertoBarrago/blast` →
    `AlbertoBarrago/impactline` (GitHub redirects the old URL
    automatically)
  - Go module path: `github.com/AlbertoBarrago/blast` →
    `github.com/AlbertoBarrago/impactline`
  - Binary/command name: `blast` → `impactline`
  - Config file: `.blast.yml` → `.impactline.yml`, no backward-compat
    fallback
  - Man pages: `blast.1`/`blast-*.1` → `impactline.1`/`impactline-*.1`
  - Homebrew: `brew install AlbertoBarrago/tap/impactline` replaces
    `brew install AlbertoBarrago/tap/blast`; the `blast` formula (only
    published for about a day) is kept in the tap, deprecated, pointing
    at the new one — same treatment `changeblast` got in v0.1.15

## [0.1.16] (2026-08-27)

### Fixed

- Installation docs said `brew install blast` after `brew tap
  AlbertoBarrago/tap`. That's broken: `blast` also names an unrelated,
  long-established `homebrew-core` formula (NCBI's BLAST bioinformatics
  tool, ~150MB), which Homebrew resolves *before* a tapped formula of
  the same name for a bare `brew install blast` — silently installing
  the wrong package. All docs now say `brew install
  AlbertoBarrago/tap/blast` (fully qualified), the only form that's
  unambiguous.

## [0.1.15] (2026-08-27)

### Changed

- **Project renamed from ChangeBlast to Blast.** The CLI binary was
  always called `blast`; the project/repository/Go module name now
  matches it. Breaking changes for existing users:
  - GitHub repository: `AlbertoBarrago/changeblast` →
    `AlbertoBarrago/blast` (GitHub redirects the old URL automatically,
    including `git clone`/`go get` against the old path)
  - Go module path: `github.com/AlbertoBarrago/changeblast` →
    `github.com/AlbertoBarrago/blast`
  - Config file: `.changeblast.yml` → `.blast.yml` (no backward-compat
    fallback; this project has no meaningful installed base yet)
  - Homebrew: `brew install AlbertoBarrago/tap/blast` replaces
    `brew install changeblast`; the old formula is kept in the tap,
    marked deprecated with a pointer to the new one, rather than
    deleted outright

## [0.1.14] (2026-08-27)

### Added

- `--explain` now works on `blast inspect <directory>` and `blast
  diff`, not just single-file `inspect`: one call per file, run
  sequentially (documented as slow-but-honest rather than guessing at
  a "reasonable" concurrency limit across three very different
  backends). `--json --explain` wraps each result the same way the
  single-file case already did.

### Changed

- `cmd/explain.go` centralizes the `--explain` machinery shared by
  `inspect` and `diff` (flag registration, provider selection,
  Finding construction, failure rendering), so the two commands can't
  drift in how they build a Finding or pick a provider.

## [0.1.13] (2026-08-27)

### Added

- `--explain-provider {ollama,claude,codex,gemini}` on `blast inspect
  --explain`: `claude`, `codex`, and `gemini` shell out to the
  respective already-installed, already-authenticated local CLI
  (`internal/ai/localcli`) instead of requiring a raw provider API key
  — whatever subscription/account already signs the user into that
  CLI is what `--explain` reuses. `ollama` remains the default.
  `--explain-model` maps to each CLI's own `--model` flag;
  `--explain-host` stays `ollama`-specific.

## [0.1.12] (2026-08-27)

### Added

- Azure DevOps Pipelines support (`internal/ci/azure`): discovers
  `azure-pipelines.yml` at the repository root as a single workflow,
  with path filters from the union of `trigger.paths.include` and
  `pr.paths.include`. A `trigger`/`pr` set to `none` is ignored; any
  other shape without `paths.include` makes the whole pipeline
  unfiltered.
- Jenkins declarative pipeline support (`internal/ci/jenkins`):
  regex-based extraction (Jenkinsfile is Groovy, not YAML) of
  `stage('Name') { ... }` blocks and their `changeset "pattern"`
  when-conditions, one workflow per stage. Scripted pipelines
  (no `pipeline { stages { ... } }` structure) yield no workflows.
- `inspect`/`diff` now check all four CI providers (GitHub Actions,
  GitLab CI, Azure DevOps, Jenkins) for relevant workflows, completing
  the originally planned v0.1 CI provider set.

## [0.1.11] (2026-08-27)

### Added

- GitLab CI support (`internal/ci/gitlab`): discovers jobs in
  `.gitlab-ci.yml` at the repository root and extracts their path
  filters from `rules[].changes` (or the older `only.changes`). A job
  with no rules, or any rule missing `changes:`, is treated as
  unfiltered, matching the GitHub Actions provider's stance. `include:`
  and `extends:` are not followed. `inspect`/`diff` now check both
  GitHub Actions and GitLab CI for relevant workflows.

## [0.1.10] (2026-08-27)

### Added

- C language support (`.c`/`.h`, not C++): quoted includes
  (`#include "foo.h"`) resolved relative to the including file's own
  directory. Angle-bracket includes (`#include <stdio.h>`) are always
  treated as system/library headers and not traversed. This completes
  the originally planned v0.1 language set (JS/TS, Go, Python, Java, C).

## [0.1.9] (2026-08-27)

### Added

- Java language support: plain imports (`import a.b.C;`), type
  wildcard imports (`import a.b.*;`), and static imports including
  static wildcard (`import static a.b.C.member;`). Each file's own
  `package a.b;` declaration derives its source root (no
  repository-wide manifest like Go/JS have); standard library and
  Maven/Gradle dependency imports are recorded as external.

### Fixed

- `make man-check` failed every day after the man pages were last
  committed, even with no real content change, because it compared
  cobra/doc's auto-generated "HISTORY" date line as if it were content
  drift. That line is now excluded from the comparison.

## [0.1.8] (2026-08-26)

### Added

- Python language support: plain imports, from-imports (aliased and
  parenthesized multi-line forms), and relative imports
  (`from . import x`, `from ..pkg import y`). The repository root is
  treated as the sole `sys.path` entry; standard library and
  third-party imports are recorded as external, same treatment as
  JS/TS bare specifiers and Go standard-library imports.

### Fixed

- `inspect`/`diff`/`graph` CLI messages said "JS/TS module" even for
  Go or Python targets, stale wording left over from before Go
  support was added. Now language-neutral ("recognized module").

## [0.1.7] (2026-08-26)

### Added

- `.changeblast.yml`: optional per-repository config file, read from
  the repository root, overriding the default critical-path keyword
  list (`criticalPaths`) and the Git history window
  (`historyWindow.days`, `historyWindow.maxCommits`). Read by `inspect`,
  `diff`, and `history`; both keys are independently optional and fall
  back to the v0.1 built-in defaults when absent.

## [0.1.6] (2026-08-26)

### Fixed

- `--explain` output showed literal Markdown asterisks (`**bold**`)
  when the model ignored the "plain prose only" prompt instruction.
  The prompt now explicitly forbids Markdown formatting, and terminal
  rendering additionally strips any bold/code/header/bullet markup
  defensively (`output.StripMarkdown`), converting `**bold**` spans to
  ANSI bold when color is enabled rather than printing raw asterisks.

## [0.1.5] (2026-08-26)

### Fixed

- `--explain` without `--explain-model` failed with a raw 404 error and
  no "try `ollama pull`" hint when the default model (`llama3.2`)
  wasn't pulled: Ollama returns that hint-worthy error inside a 404
  body, not just a 200 response, and the previous code only checked for
  it on 200. The hint now fires regardless of status code.
- The default model now falls back automatically to a model that is
  actually pulled (the first one Ollama reports) when `llama3.2` isn't
  available, instead of always failing until the user passes
  `--explain-model` by hand. An explicitly passed `--explain-model` is
  never silently swapped.
- `blast doctor` now lists the names of pulled Ollama models, not just
  a count, so it's obvious what to pass to `--explain-model`.

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
