# ChangeBlast Usage Guide

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap AlbertoBarrago/tap
brew install changeblast
```

### From source

Build from source (Go 1.22+ required):

```bash
git clone https://github.com/AlbertoBarrago/changeblast
cd changeblast
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

- **Target** — the file you asked about.
- **Direct impact** — files that directly `import`/`require` the target.
- **Indirect impact** — files that depend on the target transitively
  (through a chain of direct impacts).
- **CI** — GitHub Actions workflows relevant to the target.
- **Git history** — churn and co-change frequency within the history
  window.
- **Risk** — an explainable score with a line-by-line breakdown.

## Commands (v0.1)

### `blast inspect <path>`

Canonical form. Runs the full analysis pipeline for a single file.

```bash
blast inspect src/auth/token.ts
blast inspect src/auth/token.ts --json
blast inspect src/auth/token.ts --fail-on high
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
  HIGH — 82/100
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
or exceeds the given threshold — see Exit codes below.

### `blast diff [<ref>]`

Runs the same full analysis for every JS/TS file changed between `<ref>`
and the current working tree, including uncommitted and untracked
changes. Default `<ref>` is `HEAD` (uncommitted changes only).

```bash
blast diff
blast diff HEAD~1
blast diff main --json
blast diff --fail-on high
```

Text output renders each changed file's full inspect report, separated
by `---`. `--json` emits an array of the same per-file JSON shape as
`inspect --json`. `--fail-on` gates on the worst risk level across all
changed files.

Analyzing a single isolated commit (no working tree) is out of scope for
v0.1.

### `blast graph <path>`

Shows the file's direct dependencies and dependents, one level in each
direction — a narrower, structural view than `inspect`'s downstream
impact analysis.

```bash
blast graph src/auth/token.ts
blast graph src/auth/token.ts --json
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

### `blast history <path>`

Shows Git churn and co-change frequency for a file, computed over the
last 90 days or the last 200 commits touching the file (whichever is
smaller — see `docs/architecture.md`). This is the same signal shown in
`inspect`'s "Git history" section, standalone.

```bash
blast history src/auth/token.ts
blast history src/auth/token.ts --json
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

Checks the local environment and current repository for ChangeBlast
compatibility (git availability, repository detection, tsconfig.json
presence, GitHub Actions workflows, git history availability). Exits
non-zero if a required check fails.

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

## Module resolution — what blast understands (v0.1)

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
- Monorepo workspaces (pnpm/yarn/npm workspaces, Nx, Turborepo) — the
  whole repository is treated as one package graph

See `docs/architecture.md` for the rationale behind these limitations.

## Risk scoring

The risk score is a sum of documented, explainable contributions — see
`docs/architecture.md` for the full weight table and the fixed
critical-path keyword list (`auth`, `payment`, `billing`, `security`).
Every point shown in a `Risk` breakdown maps to a named rule; there is no
hidden or unexplained score.

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success — analysis completed (risk threshold not exceeded, or `--fail-on` not passed) |
| `1`  | Execution error (invalid target, not a git repo, internal failure) |
| `2`  | Analysis completed but risk threshold exceeded (`inspect`/`diff` with `--fail-on <level>` only) |

Without `--fail-on`, a HIGH risk result still exits `0`.

## Known limitations

- CI relevance is currently GitHub Actions only.
- A workflow with multiple triggers where only some declare `paths` is
  treated as unfiltered (always relevant) — narrowing that correctly
  requires modeling which trigger actually fires, out of scope for v0.1.
- `.changeblast.yml` configuration (critical-path/history-window
  overrides) is not implemented yet; the v0.1 defaults are the only
  source of truth.
