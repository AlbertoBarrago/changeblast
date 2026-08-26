# ChangeBlast Usage Guide

## Installation

Build from source (Go 1.22+ required):

```bash
git clone <repo-url>
cd changeblast
go build -o blast .
```

Place the resulting `blast` binary on your `PATH`.

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

## Commands (v0.1)

### `blast inspect <path>`

Canonical form. Analyzes direct and indirect dependents of a single file.

```bash
blast inspect src/auth/token.ts
blast inspect src/auth/token.ts --json
```

`--json` emits the same information as structured JSON for scripting/CI:

```json
{
  "target": "src/auth/token.ts",
  "direct": ["src/auth/middleware.ts"],
  "indirect": ["src/api/client.ts"]
}
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
smaller — see `docs/architecture.md`).

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
presence). Exits non-zero if a required check fails.

```bash
blast doctor
```

### `blast version`

Prints the CLI version.

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

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success |
| `1`  | Execution error (invalid target, not a git repo, internal failure) |

`--fail-on <level>` (risk-threshold CI gating) is planned once the risk
engine ships; it is not implemented in v0.1.

## Roadmap commands (not yet available)

`blast diff`, `blast diff <ref>`, `blast graph <path>`,
`blast history <path>` are part of the target design but not implemented
in this version. See `docs/architecture.md` for what's scaffolded versus
what has logic behind it.
