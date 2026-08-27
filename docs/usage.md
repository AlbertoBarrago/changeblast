# Blast Usage Guide

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap AlbertoBarrago/tap
brew install blast
```

### From source

Build from source (Go 1.22+ required):

```bash
git clone https://github.com/AlbertoBarrago/blast
cd blast
go build -o blast .
```

Place the resulting `blast` binary on your `PATH`.

### Shell completion

```bash
echo 'source <(blast completion zsh)' >> ~/.zshrc   # or bash/fish, see blast completion --help
```

## Quick start

From inside a Git repository containing JavaScript/TypeScript source:

```bash
blast inspect src/auth/token.ts
```

This prints:

- **Target**: the file you asked about.
- **Direct impact**: files that directly `import`/`require` the target.
- **Indirect impact**: files that depend on the target transitively
  (through a chain of direct impacts).
- **CI**: GitHub Actions workflows relevant to the target.
- **Git history**: churn and co-change frequency within the history
  window.
- **Risk**: an explainable score with a line-by-line breakdown.

## Commands (v0.1)

### `blast inspect [path]`

Canonical form. Runs the full analysis pipeline for a single file.
`<path>` defaults to `.` (the current directory) if omitted.

```bash
blast inspect src/auth/token.ts
blast inspect src/auth/token.ts --json
blast inspect src/auth/token.ts --fail-on high
blast inspect src/auth/token.ts --output report.txt
```

```
Target
  src/auth/token.ts

Direct impact
  auth.middleware.ts

Indirect impact
  checkout.ts

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

`--json` emits the same information as structured JSON for scripting/CI
(`target`, `direct`, `indirect`, `historyWindow`, `changes`, `coChanged`,
`relevantWorkflows`, `risk`).

`--fail-on <low|medium|high>` exits with code 2 when the risk level meets
or exceeds the given threshold: see Exit codes below.

`--output <file>` (or `-o`) writes the report to that file instead of
stdout (works with `--json` too), and disables color automatically
(color only ever applies to a real terminal).

#### `--explain`

Asks an AI provider to explain each result's risk score in natural
language, in addition to the deterministic report. Available on
`blast inspect` (single file or directory) and `blast diff`. For a
directory or a multi-file diff, this makes one call **per file**,
sequentially — can be slow; see "Known limitations" below.

```bash
blast inspect src/auth/token.ts --explain
blast inspect src/auth/token.ts --explain --explain-model llama3.2
blast inspect src/auth/token.ts --explain --explain-host http://localhost:11434
blast inspect . --explain            # one call per file in the directory
blast diff --explain                 # one call per changed file

# local CLI providers instead of Ollama — reuses whatever
# subscription/account already authenticates the CLI on this machine,
# no API key needed
blast inspect src/auth/token.ts --explain --explain-provider claude
blast inspect src/auth/token.ts --explain --explain-provider codex
blast inspect src/auth/token.ts --explain --explain-provider gemini
```

`--explain-provider` picks the backend (default `ollama`):

- `ollama` (default): requires `ollama serve` running locally. Without
  `--explain-model`, blast tries `llama3.2` first, and if that isn't
  pulled, automatically falls back to whichever model Ollama reports as
  available (`blast doctor` lists them) rather than failing.
  `--explain-host` only applies here.
- `claude`, `codex`, `gemini`: require the respective CLI (`claude`,
  `codex`, `gemini`) installed on `PATH` and already signed in — blast
  never manages credentials for these, it shells out to them exactly
  as you would from your own terminal. `--explain-model` maps to each
  CLI's own `--model` flag when set, otherwise its own default model
  applies. `--explain-host` is ignored for these three.

Whichever provider is chosen, a model passed explicitly via
`--explain-model` is always used as-is, never silently swapped
(Ollama's own auto-fallback above only kicks in when `--explain-model`
is omitted). No network call or subprocess is started unless
`--explain` is passed. If the provider is unreachable, unauthenticated,
or not installed, the deterministic report still prints normally and a
warning is shown instead of the explanation; the command's exit code is
unaffected by an explanation failure. `--json --explain` wraps the
response as `{"analysis": {...}, "explanation": "..."}` instead of the
flat shape used without `--explain`, to keep the default `--json`
output unchanged for existing scripts.

#### `blast inspect <directory>`

Given a directory instead of a file (including `.` for the whole
repository), every module inside it is analyzed and reported as a
compact, risk-sorted summary instead of one full report per file:

```bash
blast inspect .
blast inspect src/auth
```

```
Analyzed src/auth (2 files)

HIGH   82/100  src/auth/token.ts                                  (14 downstream)
MEDIUM 45/100  src/auth/middleware.ts                             (6 downstream)

1 HIGH, 1 MEDIUM, 0 LOW
```

`--json` on a directory target emits an array using the same per-file
JSON shape as `blast diff --json`.

### `blast diff [<ref>]`

Runs the same full analysis for every JS/TS file changed between `<ref>`
and the current working tree, including uncommitted and untracked
changes. Default `<ref>` is `HEAD` (uncommitted changes only).

```bash
blast diff
blast diff HEAD~1
blast diff main --json
blast diff --fail-on high
blast diff --output report.txt
```

Text output renders each changed file's full inspect report, separated
by `---`. `--json` emits an array of the same per-file JSON shape as
`inspect --json`. `--fail-on` gates on the worst risk level across all
changed files.

Analyzing a single isolated commit (no working tree) is out of scope for
v0.1.

### `blast graph <path>`

Shows the file's direct dependencies and dependents, one level in each
direction: a narrower, structural view than `inspect`'s downstream
impact analysis.

```bash
blast graph src/auth/token.ts
blast graph src/auth/token.ts --json
blast graph src/auth/token.ts --output graph.txt
```

### `blast <path>` (alias)

A convenience shortcut for `blast inspect <path>`, available on the root
command. Resolution order:

1. If `<path>` matches a registered subcommand name exactly (`diff`,
   `graph`, `doctor`, `history`, `version`, `completion`, `inspect`), it
   is treated as that subcommand.
2. Otherwise, if `<path>` exists in the working tree, it is treated as
   `blast inspect <path>`.
3. Otherwise, blast errors with `unknown command or path: "<path>"`.

**Prefer the canonical `blast inspect <path>` form in scripts and CI**
to avoid any ambiguity with a file that happens to share a name with a
subcommand.

### `blast history [path]`

Shows Git churn and co-change frequency for a file or directory,
computed over the last 90 days or the last 200 commits touching it
(whichever is smaller, see `docs/architecture.md`). This is the same
signal shown in `inspect`'s "Git history" section, standalone.
`<path>` defaults to `.` (the current directory) if omitted.

```bash
blast history src/auth/token.ts
blast history src/auth/token.ts --json
blast history src/auth/token.ts --output history.txt
```

```
Target
  src/auth/token.ts

Git history
  7 significant changes (last 90 days)
  3 frequently co-changed modules

Frequently co-changed
  src/auth/middleware.ts (5 times)
  src/api/client.ts (3 times)
  src/session.service.ts (2 times)
```

A co-changed file is counted as "frequent" once it appears alongside the
target in at least 2 commits within the window.

### `blast doctor`

Checks the local environment and current repository for Blast
compatibility (git availability, repository detection, tsconfig.json
presence, GitHub Actions workflows, git history availability, and
whether a local Ollama daemon is reachable). Exits non-zero if a
required check fails; Ollama reachability is informational only, since
it's optional and only needed for `--explain`. This is the one case
where `blast doctor` makes a network call, and it is always to
localhost/`$OLLAMA_HOST`, never a remote host.

```bash
blast doctor
```

### `blast version`

Prints the CLI version.

### Shell completion and man page

```bash
blast completion bash|zsh|fish
man blast
```

The man page is generated from Cobra command metadata (`make man`); see
`docs/architecture.md`.

## Module resolution: what blast understands (v0.1)

blast resolves:

- Relative ESM imports: `import { x } from './x'`, `export { y } from '../y'`
- CommonJS: `require('./x')`
- `tsconfig.json` `baseUrl` / `paths` aliases (single tsconfig.json at the
  repository root or nearest ancestor directory)
- Extension resolution (`./x` -> `x.ts`, `x.tsx`, `x.js`, ...) and index
  resolution (`./x` -> `x/index.ts`, ...)

blast does **not** (yet) resolve:

- Bare package imports into `node_modules` (recorded as external, not
  traversed)
- package.json `exports`/`imports` maps
- Dynamic `import()` expressions (recorded as evidence, not traversed)
- Barrel re-exports beyond one level
- Monorepo workspaces (pnpm/yarn/npm workspaces, Nx, Turborepo): the
  whole repository is treated as one package graph

For Go files, blast resolves:

- Single-line (`import "fmt"`) and grouped (`import ( "a"; "b" )`)
  imports, including aliased/blank/dot forms
- Imports whose path is the current module (from `go.mod`) or a
  subpackage of it, one edge per file in the target package

blast does **not** (yet) resolve, for Go:

- Standard library or external module imports (recorded as external)
- Go workspaces (`go.work`, multi-module repositories)

For Python files, blast resolves:

- Plain imports (`import a.b.c`, aliased and comma-separated forms) and
  from-imports (`from a.b import c`, aliased and parenthesized
  multi-line forms), including relative imports (`from . import x`,
  `from ..pkg import y`)
- The repository root is treated as the sole entry on `sys.path`

blast does **not** (yet) resolve, for Python:

- Standard library or third-party imports (recorded as external)
- `src/` layout auto-detection, virtualenv/`site-packages`, or
  `PYTHONPATH`
- Wildcard imports (`from x import *`)

For Java files, blast resolves:

- Plain imports (`import a.b.C;`), type wildcard imports
  (`import a.b.*;`), and static imports including static wildcard
  (`import static a.b.C.member;`, `import static a.b.C.*;`)
- Each file's own `package a.b;` declaration is used to derive its
  source root (no repository-wide manifest like `go.mod`/`tsconfig.json`
  to anchor against); this assumes a standard Maven/Gradle layout
  (`src/main/java/a/b/Foo.java` declaring `package a.b;`)

blast does **not** (yet) resolve, for Java:

- JDK standard library or third-party (Maven/Gradle dependency) imports
  (recorded as external)
- Maven/Gradle multi-module builds with cross-module dependencies
- Java 15+ text blocks (`"""..."""`) in the comment/string stripper

For C files (`.c`/`.h` only, not C++), blast resolves:

- Quoted includes (`#include "foo.h"`), relative to the including
  file's own directory only

blast does **not** (yet) resolve, for C:

- Angle-bracket includes (`#include <stdio.h>`), always treated as a
  system/library header
- Compiler include paths (`-I` flags, `CPATH`) or any build system
  (Make/CMake) awareness
- Preprocessor conditionals (`#ifdef`/`#endif`): an `#include` inside a
  disabled branch is still extracted

See `docs/architecture.md` for the rationale behind these limitations.

## Risk scoring

The risk score is a sum of documented, explainable contributions: see
`docs/architecture.md` for the full weight table and the default
critical-path keyword list (`auth`, `payment`, `billing`, `security`).
Every point shown in a `Risk` breakdown maps to a named rule; there is no
hidden or unexplained score.

## Repository configuration (`.blast.yml`)

An optional `.blast.yml` at the repository root overrides two v0.1
defaults, read by `inspect`, `diff`, and `history`:

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

Both keys are optional independently; an absent file, or an absent key
within it, falls back to the built-in default. `criticalPaths` replaces
the default keyword list entirely when set (it does not merge with it).
See `docs/architecture.md` for the full lookup rule (repository root
only, no upward directory walk) and how these values flow into the risk
score and Git history window.

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: analysis completed (risk threshold not exceeded, or `--fail-on` not passed) |
| `1`  | Execution error (invalid target, not a git repo, internal failure) |
| `2`  | Analysis completed but risk threshold exceeded (`inspect`/`diff` with `--fail-on <level>` only) |

Without `--fail-on`, a HIGH risk result still exits `0`.

## Known limitations

- CI relevance covers GitHub Actions, GitLab CI, Azure DevOps
  Pipelines, and Jenkins declarative pipelines.
- A GitHub Actions workflow with multiple triggers where only some
  declare `paths`, a GitLab CI job with multiple `rules:` entries where
  only some declare `changes:`, or an Azure Pipelines `trigger`/`pr`
  lacking `paths.include`, is treated as unfiltered (always relevant);
  narrowing that correctly requires modeling which trigger/rule
  actually applies, out of scope for v0.1.
- GitLab CI's `include:` (splitting a pipeline across multiple files)
  and `extends:` (inheriting a hidden job's `rules:`) are not followed;
  only `.gitlab-ci.yml`'s own top-level jobs are considered.
- Azure DevOps: only the default `azure-pipelines.yml` location is
  checked (a pipeline renamed/relocated via the project UI is not
  discovered).
- Jenkins: only declarative pipelines are supported (a scripted
  pipeline has no `stage()`/`changeset` structure to extract); stage
  boundaries are approximated by text position, not real brace
  balancing, since `Jenkinsfile` is Groovy and v0.1 uses a regex-based
  extraction rather than a real parser.
- `--explain` on `blast inspect <directory>` or `blast diff` runs one
  call per file, sequentially, with no concurrency limit — expect it to
  take roughly (call latency) × (file count). There is no batching or
  parallelism across the three backends (a local daemon and two agent
  CLIs, each with its own rate limits and startup cost).
