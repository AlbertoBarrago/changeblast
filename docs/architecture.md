# ChangeBlast Architecture

## Pipeline

```
CLI (cmd/)
  -> Target Resolver (internal/repository)
  -> Repository Scanner (internal/repository)
       -> Language Analyzer (internal/analyzer/*)
       -> Git Analyzer (internal/git)
       -> CI Analyzer (internal/ci)
  -> Dependency Graph (internal/graph)
  -> Impact Engine (internal/impact)
  -> Risk Engine (internal/risk)
  -> Output (internal/output): text | json
```

Each stage only depends on the stage before it through plain data structures
(`graph.Graph`, `impact.Result`, ...). The CLI layer (`cmd/`) is the only
package that wires everything together; no package below it imports Cobra
or does terminal formatting. This is what keeps `internal/impact` and
`internal/risk` usable from a future `--json` consumer, a test, or a
different frontend without dragging in CLI concerns.

## Why a regex-based JS/TS analyzer instead of a full parser

v0.1 extracts imports with a small set of regular expressions
(`internal/analyzer/javascript`) rather than a real AST parser (e.g. a Go
TypeScript parser or shelling out to `tsc`/`swc`). This is a deliberate
tradeoff for the first vertical slice:

- **Zero heavy dependencies.** A conformant TS/TSX parser in Go is either a
  large dependency or requires embedding a JS runtime. Neither fits
  "single standalone binary, fast, local-first."
- **Good enough for the target statement shapes.** The regexes cover
  `import ... from '...'`, bare `import '...'`, `export ... from '...'`,
  `require('...')`, and `import('...')`, which is the overwhelming
  majority of real-world import syntax.
- **Known failure mode.** A specifier-shaped string inside a string or
  template literal that isn't a comment (e.g. `const s = "import x from
  'y'"`) can produce a false positive. Comments (`//`, `/* */`) are
  stripped before matching to eliminate the most common false-positive
  source. This is documented here rather than silently accepted: if this
  becomes a real problem, the fix is a proper tokenizer/lexer pass, not a
  bigger regex.

If/when this becomes a real limitation (reported false positives on real
repos), the next step is a minimal hand-written lexer that only needs to
distinguish code from strings/comments/template literals (still far
short of a full parser) rather than pulling in a third-party TS parser.

## Module resolution scope (v0.1)

Implemented in `internal/repository` (`resolve.go`, `tsconfig.go`):

- Relative specifiers (`./x`, `../x`) resolved against the importing file's
  directory.
- Extension resolution: `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs`.
- Index resolution: `x/index.ts`, `x/index.tsx`, `x/index.js`, `x/index.jsx`.
- `tsconfig.json` `compilerOptions.baseUrl` and `paths` (single wildcard
  per pattern), found by walking upward from the repository root to the
  nearest ancestor containing a `tsconfig.json`.

Explicitly out of scope (recorded as external/unresolved, not traversed):

- Resolution into `node_modules`: bare specifiers are external dependency
  edges only when they match a tsconfig alias; otherwise they are dropped.
- package.json `exports`/`imports` map resolution.
- Dynamic `import()` expressions: recorded as evidence (`Dynamic: true`
  on the raw import) but not resolved or traversed.
- Barrel file re-export flattening beyond one level.
- Monorepo workspace resolution (pnpm/yarn/npm workspaces, Nx, Turborepo).
  A repository is one single package graph in v0.1.

These are known limitations, not bugs: the goal of v0.1 is a correct
answer within a clearly stated scope, not a best-effort guess outside it.

## Go module resolution scope (v0.1)

Implemented in `internal/repository` (`gomod.go`, `goresolve.go`) and
`internal/analyzer/golang`:

- Single-line (`import "fmt"`) and grouped (`import ( "a"; b "c/d" )`)
  imports, including aliased, blank (`_`), and dot (`.`) forms (the
  local identifier is irrelevant to blast-radius edges, so it is parsed
  but not tracked).
- Only imports whose path is the current module (from `go.mod`) or a
  subpackage of it are resolved. Standard library and external module
  imports are recorded as external, exactly like JS/TS bare specifiers
  into `node_modules`: not traversed into the module cache or GOPATH.
- A single `go.mod` at the repository root or nearest ancestor, the same
  pattern as `tsconfig.json` (`FindGoModule` mirrors `FindTSConfig`).

**Structural difference from JS/TS that shapes the graph:** a Go import
targets a *package* (a directory, potentially several files), not a
single file. `GoResolver.Resolve` reflects this directly in its
signature (it returns `[]string`, not `(string, bool)` like the JS/TS
resolver), and `Scanner` adds one edge per file in the target package.
This means editing any file in a Go package is treated as equally
impactful to that package's importers, which is coarser than JS/TS's
per-file precision but matches Go's actual compilation unit.

Explicitly out of scope: Go workspaces (`go.work`, multi-module
repositories). A repository is one single module in v0.1, same
limitation category as JS/TS monorepo workspaces above.

## Python module resolution scope (v0.1)

Implemented in `internal/repository` (`pyresolve.go`) and
`internal/analyzer/python`:

- Plain imports (`import a.b.c`, `import a.b.c as x`, comma-separated
  forms) and from-imports (`from a.b import c`, aliased and
  parenthesized multi-line forms), including relative imports
  (`from . import x`, `from ..pkg import y`).
- The repository root is treated as the sole entry on `sys.path`: an
  absolute import resolves against it directly, with no `src/` layout
  auto-detection, no virtualenv/`site-packages` resolution, and no
  `PYTHONPATH` support. A relative import's dot count is treated as
  "how many directory levels to go up from the importing file's own
  directory," which doesn't distinguish a regular module from an
  `__init__.py` the way Python itself does — a documented approximation,
  not a bug, consistent with the rest of v0.1's stated scope.
- A `from <module> import <name>` specifier is ambiguous without
  deeper analysis: `<name>` may be a submodule (a file) or an attribute
  defined inside `<module>` (not a file on its own). `PythonResolver.Resolve`
  tries the full path as a submodule first and falls back to `<module>`
  (i.e. its `__init__.py`) if that doesn't exist, rather than guessing
  from syntax alone.
- Only imports that resolve to a file under the repository root are
  traversed; standard library and third-party imports are recorded as
  external, exactly like a JS/TS bare specifier into `node_modules` or a
  Go standard-library import.

Explicitly out of scope: wildcard imports (`from x import *`, no name to
resolve and no re-export flattening), namespace package edge cases, and
`sys.path`/`PYTHONPATH` manipulation.

## Java module resolution scope (v0.1)

Implemented in `internal/repository` (`javaresolve.go`) and
`internal/analyzer/java`:

- Plain imports (`import a.b.C;`), type wildcard imports
  (`import a.b.*;`), and static imports including static wildcard
  (`import static a.b.C.member;`, `import static a.b.C.*;`).
- Unlike Go (anchored on `go.mod`) or JS/TS (anchored on
  `tsconfig.json`), Java v0.1 has no repository-wide manifest to derive
  resolution from. Each importing file's own `package a.b;` declaration
  is used instead, to derive that file's source root by walking up one
  directory per package segment from the file's own path. This assumes
  the conventional layout where a file's directory path suffix matches
  its package name (e.g. `src/main/java/a/b/Foo.java` declaring
  `package a.b;`), true for any standard Maven/Gradle layout — the
  documented v0.1 scope, same category of assumption as JS/TS's
  relative-import-only resolution. A file with no `package` declaration
  (the default package) is assumed to live at its own source root.
- A static import's imported name is always a member (a field or
  method) of the class named by every segment before it — unlike
  Python's `from X import Y`, this is structurally unambiguous from
  Java's grammar alone, so `JavaResolver.Resolve` always drops the last
  segment for a static import, no fallback/guessing needed.
- A type wildcard import (`import a.b.*;`) resolves to every other
  `.java` file in the target package directory, matching the
  package-level granularity Go uses for its own imports (one specifier,
  potentially several files) rather than JS/TS's per-file precision.
- Only imports that resolve to a file under the repository are
  traversed; JDK standard library and third-party imports are recorded
  as external, exactly like a JS/TS bare specifier into `node_modules`
  or a Go standard-library import.

Explicitly out of scope: Java 15+ text blocks (`"""..."""`, not
specially handled by the comment/string stripper, a rare source of
false positives), Maven/Gradle multi-module builds with cross-module
dependencies, and annotation processing.

## C module resolution scope (v0.1)

Implemented in `internal/repository` (`cresolve.go`) and
`internal/analyzer/c`:

- Only quoted includes (`#include "foo.h"`) are extracted as raw
  imports at all; angle-bracket includes (`#include <stdio.h>`) are
  always a system/library header and are skipped at extraction time,
  since there's nothing else the quoted-vs-angle distinction could mean
  in v0.1's scope — simpler than deciding it during resolution the way
  Python's from-import ambiguity is.
- A quoted include resolves relative to the including file's own
  directory only (`../auth/token.h` from `src/api/client.c` resolves
  against `src/api`, same as a JS/TS relative import). v0.1 has no
  awareness of compiler include paths (`-I` flags, `CPATH`) or any
  build system (Make/CMake); an include that isn't resolvable relative
  to its including file is recorded as external/unresolved.
- Preprocessor conditionals (`#ifdef`/`#endif`) are not evaluated: an
  `#include` inside a disabled branch is still extracted. A documented
  over-approximation, not a bug, consistent with the CI analyzer's
  "false relevant is safer than a missed one" stance.

Explicitly out of scope: macro expansion, any build-system awareness,
and C++ (`.cpp`/`.hpp`/`.cc`) — v0.1 handles `.c`/`.h` only.

## The dependency graph

`internal/graph` is a plain directed graph keyed by absolute file path,
with both forward (`importer -> imported`) and reverse
(`imported -> importer`) adjacency maintained simultaneously. This
duplication is intentional: impact analysis (`internal/impact`) only ever
needs the reverse edges (who depends on this file), and computing that
via BFS over `reverse` is O(dependents reached), not O(all edges) per
query, which matters once `graph`/`history` commands need repeated
queries against the same scanned graph.

The graph does not enforce acyclicity. Import cycles are structurally
possible; `impact.Compute` guards against revisiting nodes so a cycle
cannot cause an infinite loop.

## Repository scanning

`internal/repository.Scanner` walks the filesystem with
`filepath.WalkDir`, skipping `node_modules`, `.git`, `dist`, `build`,
`coverage`, and `vendor` at directory-entry level (`filepath.SkipDir`) so
those subtrees are never descended into, not just filtered after reading.
Every file handled by a registered analyzer becomes a graph node even if
it has zero imports, so an isolated file can still be a valid `inspect`
target.

Each language is registered as a `languageSupport` pair: an
`analyzer.Analyzer` (extracts raw imports from text) plus a `resolve`
function (turns a raw import into zero or more absolute file paths).
`Scanner.Scan` itself contains no language-specific logic beyond
selecting which pair's `analyzer.CanHandle` matches a given file. The
`resolve` signature (`func(fromFile string, imp analyzer.RawImport)
[]string`) is deliberately generic enough to cover both "one specifier
resolves to at most one file" (JS/TS) and "one specifier resolves to a
whole package's worth of files" (Go) without either language needing
special-casing in the scanner. Adding a language means adding one
`languageSupport` entry in `NewScanner`, nothing else.

## CLI command resolution

`blast <path>` (root command, no subcommand) is a convenience alias for
`blast inspect <path>`. See the `blast --help` output (generated from
`cmd/root.go`'s `Long` description) for the exact precedence rules. This
logic intentionally lives in `cmd/root.go` only: no other package needs
to know the alias exists.

## Git analyzer and history window

`internal/git` shells out to the `git` binary (already a required
dependency, per `blast doctor`) rather than parsing `.git` internals or
using a pure-Go Git library. `git log`/`git show` output is already the
data we need (commit hashes touching a path, files changed per commit);
reimplementing that against packfiles would add real complexity for no
behavioral gain in v0.1.

All historical signals (churn, co-change frequency) are computed over a
bounded window: the last `HistoryWindowDays` (90) days, or the last
`HistoryWindowMaxCommits` (200) commits touching the file, whichever is
smaller (`git log --since=90.days.ago --max-count=200`). This is a named
constant (`internal/git/history.go`), never an unstated magic number, and
is surfaced in `--json` output as `historyWindow` and in text output as
"N significant changes (last 90 days)". Overridable per-repository via
`.changeblast.yml`'s `historyWindow.days`/`historyWindow.maxCommits`
(see "Repository configuration" below); `git.AnalyzeWithWindow` is the
entry point callers use once they've resolved the override, while
`git.Analyze` remains a thin wrapper over it with the built-in default.

Co-change frequency is computed by tallying, across the commits touching
the target file, every other file present in the same commit
(`git show --name-only`). This is O(commits in window), bounded by the
same window, so it stays cheap even on files with long histories.

## CI analyzer

`internal/ci` defines a provider-agnostic `Workflow`/`Provider`
interface; `internal/ci/github` implements it for GitHub Actions
(`.github/workflows/*.yml`). Path filters (`on.push.paths` /
`on.pull_request.paths`) are extracted with `gopkg.in/yaml.v3` rather
than regex-scraping YAML, since YAML's structure (anchors, flow vs. block
style, multi-document triggers) is not reliably regex-matchable and this
is the one place in v0.1 where a real parser is worth the dependency.

A workflow with **any** trigger lacking a `paths` filter (including a
bare trigger like `on: push`) is treated as unfiltered (relevant to
every change) because GitHub Actions doesn't let path filters narrow
which trigger fires; modeling that correctly needs to know which trigger
actually fired, which is out of scope for v0.1. This is a documented
over-approximation, not a bug: a false "relevant" is safer than a missed
one for a tool whose job is warning about blast radius.

Path filter globs (`src/auth/**`) are matched with a small hand-rolled
glob-to-regexp translator (`internal/ci/glob.go`) rather than
`filepath.Match`, because `filepath.Match` has no `**` (match across
path segments) support, which GitHub Actions path filters rely on.

`internal/ci/gitlab` implements the same `ci.Provider` interface for
GitLab CI (`.gitlab-ci.yml` at the repository root only — `include:`,
which splits a pipeline across multiple files, is not followed, the
same "single file, no cross-file resolution" scope JS/TS's
`tsconfig.json` handling uses). Every top-level key that isn't a
reserved pipeline keyword (`stages`, `variables`, `workflow`, `include`,
etc.) or a hidden/template job (GitLab's convention: a name starting
with `.`, meant to be reused via `extends:`, which v0.1 does not
follow) is treated as a job. A job's path filter is its `rules[].changes`
(the modern syntax) or `only.changes` (the older one); a job with no
rules at all, or with **any** rule in its `rules:` list lacking a
`changes:` key, is treated as unfiltered — the same over-approximation
stance as GitHub Actions above, since evaluating `if:`/`when:`
conditions to know which rule actually applies is out of scope for
v0.1. `cmd/inspect.go`'s `discoverWorkflows` runs every registered
provider and merges their results; a provider with no config file
present contributes nothing, not an error.

`internal/ci/azure` implements the interface for Azure DevOps Pipelines
(`azure-pipelines.yml` at the repository root — the conventional
default name only; Azure DevOps lets a pipeline be renamed/relocated in
the project UI, which v0.1 has no way to discover). Unlike GitHub
Actions (one workflow per file) or GitLab CI (one workflow per job), an
Azure Pipelines file declares a single pipeline, so `Discover` always
returns at most one `ci.Workflow`; its path filter is the union of
`trigger.paths.include` and `pr.paths.include`. A `trigger`/`pr` set to
`none` is disabled and contributes neither a filter nor an "unfiltered"
signal; any other shape lacking `paths.include` (including one omitted
entirely, which defaults to running on every push) makes the whole
pipeline unfiltered — same stance as the other two providers.

`internal/ci/jenkins` implements the interface for Jenkins declarative
pipelines (`Jenkinsfile` at the repository root). Unlike the YAML-based
providers above, a Jenkinsfile is Groovy, so there's no structured
parser to lean on: `stage('Name') { ... }` blocks and `changeset
"pattern"` when-conditions are extracted with the same regex-based,
comment-stripping approach v0.1 uses for its source-language analyzers
(`gopkg.in/yaml.v3` doesn't apply here at all). A stage's body is
approximated as the text between its opening `{` and the start of the
next `stage(...)` declaration, not by balancing braces — a real Groovy
parser is out of scope for v0.1, documented rather than silently
accepted as a possible misattribution source in unusually nested
pipelines. Scripted pipelines (raw Groovy without a `pipeline { stages
{ ... } }` structure) have no stages to find, so `Discover` returns no
workflows for them, not an error.

## Risk engine

`internal/risk` computes a score as a sum of independently-explained
`Entry` contributions (`internal/risk/risk.go`), never an opaque number.
Every weight is a named constant:

| Signal | Weight | Notes |
|---|---|---|
| Downstream modules | 2/module, capped at 28 | `len(direct) + len(indirect)` |
| Critical path | +20 flat | path segment matches a keyword (see below) |
| Churn (high: ≥7, medium: ≥3, low: ≥1) | +14 / +7 / +3 | only the highest tier applies |
| Frequent co-change | +12 | ≥2 files co-changed ≥2 times in the window |
| CI impact | +8 | ≥1 relevant workflow |

Total is capped at 100. Level thresholds: `HIGH` ≥60, `MEDIUM` ≥30,
`LOW` otherwise (`risk.ThresholdHigh`, `risk.ThresholdMedium`).

**Critical path** (`internal/risk/criticalpath.go`) matches path
segments case-insensitively against a keyword list, `auth`, `payment`,
`billing`, `security` by default (`risk.DefaultCriticalPathKeywords`), a
documented default, not a hidden constant. It is a known limitation of
the default list: it will false-positive (e.g. a directory literally
named "author") and false-negative on domain-specific critical code
unless a repository overrides it via `.changeblast.yml`'s
`criticalPaths` (see "Repository configuration" below). `MatchCriticalPath`
takes the keyword list as a parameter rather than reading a package
global, precisely so the resolved (default-or-override) list can be
passed in by the caller.

The risk engine only consumes plain data (`risk.Input`) computed by
`inspectTarget` in `cmd/inspect.go`: it has no dependency on impact,
git, ci, or config packages directly, keeping it testable in isolation
and reusable from `blast diff`.

## Repository configuration (`.changeblast.yml`)

`internal/config` loads an optional `.changeblast.yml` from the
repository root only, no upward directory walk (unlike `tsconfig.json`
or `go.mod`, this is project-level configuration, not something that
varies per subpackage). A missing file resolves to a zero-value
`Config`, not an error, so every caller falls back to the same built-in
defaults whether the file is absent or simply doesn't set a given key:

```yaml
criticalPaths:
  - auth
  - payment
  - billing
  - security
historyWindow:
  days: 90
  maxCommits: 200
```

Both keys are independently optional. `Config.CriticalPathsOr`,
`HistoryWindowDaysOr`, and `HistoryWindowMaxCommitsOr` resolve a field to
its override if set, or the given built-in default otherwise; callers
(`cmd/inspect.go`, `cmd/diff.go`, `cmd/history.go`) load the config once
per invocation and thread the resolved values into
`risk.Input.CriticalPathKeywords` and `git.AnalyzeWithWindow`, rather
than `internal/risk`/`internal/git` reading the file themselves, keeping
those packages free of any dependency on `internal/config`.

## `blast diff` and CI gating

`blast diff [<ref>]` (`cmd/diff.go`) computes `git diff --name-only <ref>`
plus untracked files (`git ls-files --others --exclude-standard`, since
`git diff` does not report new untracked files) against the working
tree, then runs the same `inspectTarget` pipeline used by `blast inspect`
on each changed file that resolves to a graph node. Files that aren't
recognized JS/TS modules (config files, deleted files) are skipped rather
than aborting the whole diff.

`--fail-on <level>` on `inspect` and `diff` returns a `failOnError`
(`cmd/inspect.go`), which `Execute()` in `cmd/root.go` maps to exit code
2 per the documented exit code contract; this is the only path to a
non-zero, non-1 exit code in the CLI.

## `blast inspect <directory>`

When the target resolved by `resolveTarget` is a directory rather than a
file, `runInspect` (`cmd/inspect.go`) switches to
`runInspectDirectory`: it scans once (`buildGraph`), filters the
resulting graph's nodes to those inside the directory
(`isWithinDir`, a `filepath.Rel`-based containment check), runs
`inspectWithGraph` per file, and renders the results with
`output.RenderSummary` (a compact, one-line-per-file, risk-sorted
report) instead of `RenderInspectFull`'s per-file detail view, which
would be unusable across dozens or hundreds of files. `blast diff`'s
`--json` shape (`[]output.InspectFullJSON`) is reused for directory
`--json` output too, so both "many files at once" commands produce the
same structured shape.

## `--output`/`-o`

`cmd/outputflag.go` provides `addOutputFlag` and `openOutputTarget`,
shared by `inspect`, `diff`, `graph`, and `history`. An empty path (the
default) keeps writing to the command's stdout; a non-empty path creates
(truncating) that file and every renderer writes to it instead,
including the JSON encoders, so `--json --output x.json` works
uniformly. Color output disables itself automatically for a file target,
since `colorEnabled` only enables ANSI codes for a `*os.File` that is
also a character device (a regular file on disk never is).

## Man page generation

`docs/*.1` are generated, not hand-authored, via `cobra/doc`'s
`GenManTree` (`tools/gendocs/main.go`), driven by `make man`. They are
committed for `man blast` to work after install. `make man-check`
regenerates and diffs against the committed files, intended to run in CI
to catch drift between command help text and the committed man pages.

## Release and distribution

`.goreleaser.yml` builds `blast` for darwin/linux/windows ×
amd64/arm64 and publishes a Homebrew formula (`brews:`, still the
functional, documented way to publish a CLI formula as of GoReleaser
v2.18, despite an upstream deprecation notice pointing at the newer
`homebrew_casks` model; revisit if `brews` is actually removed) to
`AlbertoBarrago/homebrew-tap`, a tap shared with the author's other CLI
tools. Publishing to that tap requires a `HOMEBREW_TAP_GITHUB_TOKEN`
repository secret (a PAT with `repo` scope on the tap); the default
`GITHUB_TOKEN` from Actions can only write to the repository the
workflow runs in. `.github/workflows/release.yml` runs GoReleaser on
`v*` tag pushes; `v0.1.0` through `v0.1.3` have been released this way.

## AI explanation layer (`--explain`)

`internal/ai` defines a provider-agnostic `Provider` interface around a
`Finding` (the already-computed deterministic result: impact counts,
risk breakdown, history, CI relevance), plus `BuildExplainPrompt`, the
single prompt-construction function every provider shares so the
instructions a model receives ("don't invent a score", "plain prose
only") can't drift between implementations.

`internal/ai/ollama` calls a local Ollama daemon's `/api/generate`
endpoint over plain `net/http` (no new dependency needed for a
JSON-over-HTTP API) — the default provider, and the only one that never
leaves the local machine even at the network-call level.

`internal/ai/localcli` implements the same interface against three
already-installed, already-authenticated local CLIs instead of a raw
provider API: Claude Code (`claude -p ... --output-format text`), Codex
(`codex exec ... --output-last-message <file>`, since its stdout also
carries progress/log lines with no plain-text-only flag equivalent to
Claude Code's), and Gemini (`gemini -p ...`). This is deliberate: a
user who already has one of these tools set up (logged into their own
subscription/account) gets `--explain` working with zero extra
configuration — no API key to obtain, no environment variable to set,
matching this project's "the user brings their own already-set-up local
tool" stance for Ollama itself, just extended to agent CLIs rather than
a model-serving daemon. ChangeBlast never manages credentials for any
of the three; it shells out to them exactly as the user would from
their own terminal (`exec.Command`, not a shell string, so there is no
injection surface from the Finding data passed as a prompt argument).
A missing binary produces a clear "not found on PATH" error rather than
a bare exec failure; a non-zero exit surfaces the tool's own stderr, so
an unauthenticated CLI's actual diagnostic reaches the user. Selected
via `--explain-provider {ollama,claude,codex,gemini}` on `inspect`
(`ollama` remains the default); `--explain-model` maps to each CLI's
own `--model` flag when set, otherwise the tool's own default model
applies. `--explain-host` is `ollama`-specific and ignored by the
other three.

Three constraints shaped the design, all enforced structurally rather
than by convention alone:

- **Input only, never output.** `ai.Finding` has no method or field a
  provider could use to write back into the analysis; `Provider.Explain`
  returns a `string`, full stop. The risk score a user sees is always
  the one `internal/risk` computed, never anything the model produced,
  keeping "deterministic by default" intact even with explanation on.
- **Off by default, zero network calls (or subprocess spawns)
  otherwise.** `--explain` gates the entire code path in
  `cmd/inspect.go`'s `maybeExplain`; without it, no provider —
  `ollama.New` or any `internal/ai/localcli` constructor — is ever even
  constructed. `blast doctor`'s Ollama reachability check is the one
  exception, and it is explicitly called out as such (see below) since
  it does make a local network call on every `doctor` run.
- **A failed explanation is never fatal.** `renderExplanation` prints a
  warning and the deterministic report stands on its own; exit codes
  and `--fail-on` gating are computed purely from the risk score,
  unaffected by whether `--explain` succeeded.

**Why Ollama first, not a raw OpenAI/Anthropic-compatible API:** it
runs on the user's machine, consistent with "source code never leaves
the local machine by default": though note the request body only ever
contains the already-computed Finding summary (file path, counts, risk
reasons), never source code, so this constraint is about
principle-consistency, not about protecting sensitive request content
specifically.

**Why local CLIs (`localcli`) rather than a raw provider API for the
opt-in alternatives:** a direct OpenAI/Anthropic/Gemini API integration
would need ChangeBlast to accept and manage an API key — a new secret
for the user to obtain and store just for this one feature. Shelling
out to a CLI the user already has installed and signed into sidesteps
that entirely: whatever subscription/account already authenticates
`claude`/`codex`/`gemini` on their machine is what `--explain` reuses,
no new credential surface added by this project. A raw provider-API
integration (bring-your-own-key) remains a possible future addition if
someone wants `--explain` without any of these CLIs installed, but
isn't planned as of v0.1 — the three CLIs above already cover "opt into
a cloud model" for the overwhelming majority of users who'd want that.

**`--explain` on `blast diff` and `blast inspect <directory>`:** each
call is a real (often several-second, sometimes tens of seconds)
round trip, and both commands run it once per file, sequentially — no
concurrency limit or batching. This is a deliberate, documented
tradeoff rather than a solved "reasonable parallelism" problem: with
three very different backends (a local daemon vs. two agent CLIs, each
with its own rate limits and startup cost), a single hardcoded
concurrency number would be right for none of them. Both commands warn
about this in their own `--help` text ("can be slow"); a user who wants
`--explain` on a large `diff` or directory accepts that cost knowingly.
`cmd/explain.go` centralizes the shared machinery (`explainFlags`,
`newExplainProvider`, `findingFor`, `explainResult`, `renderExplanation`,
`explainedJSON`) so `inspect` and `diff` can't drift in how they build
a `Finding`, pick a provider, or render a failure — each command only
owns its own flag registration (cobra flags are per-command) and the
per-target loop that calls `explainResult`.

`blast doctor` probes `$OLLAMA_HOST` (or `http://localhost:11434`) with
a 500ms-timeout GET to `/api/tags`, reporting reachability and pulled
model count as an informational (not required) check.

## What's not implemented yet

- Additional language analyzers beyond JS/TS, Go, Python, Java, and C,
  and additional CI providers beyond GitHub Actions, GitLab CI, Azure
  DevOps, and Jenkins (both originally planned v0.1 sets are now
  complete; see the module-resolution sections above for why each
  language needed its own explicit scope decision, not just a new
  regex). The `analyzer.Analyzer` and `ci.Provider` interfaces exist
  specifically so more of either can be added without touching the
  core pipeline.
- A raw OpenAI/Anthropic/Gemini API integration (bring-your-own-key) as
  an alternative to the CLI-based `localcli` providers.
