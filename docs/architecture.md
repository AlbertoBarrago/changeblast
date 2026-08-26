# ChangeBlast Architecture

## Pipeline

```
CLI (cmd/)
  -> Target Resolver (internal/repository)
  -> Repository Scanner (internal/repository)
       -> Language Analyzer (internal/analyzer/*)
       -> Git Analyzer (internal/git)          [not yet implemented]
       -> CI Analyzer (internal/ci)            [not yet implemented]
  -> Dependency Graph (internal/graph)
  -> Impact Engine (internal/impact)
  -> Risk Engine (internal/risk)               [not yet implemented]
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
  `require('...')`, and `import('...')` — which is the overwhelming
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
distinguish code from strings/comments/template literals — still far
short of a full parser — rather than pulling in a third-party TS parser.

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

- Resolution into `node_modules` — bare specifiers are external dependency
  edges only when they match a tsconfig alias; otherwise they are dropped.
- package.json `exports`/`imports` map resolution.
- Dynamic `import()` expressions — recorded as evidence (`Dynamic: true`
  on the raw import) but not resolved or traversed.
- Barrel file re-export flattening beyond one level.
- Monorepo workspace resolution (pnpm/yarn/npm workspaces, Nx, Turborepo).
  A repository is one single package graph in v0.1.

These are known limitations, not bugs — the goal of v0.1 is a correct
answer within a clearly stated scope, not a best-effort guess outside it.

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

## CLI command resolution

`blast <path>` (root command, no subcommand) is a convenience alias for
`blast inspect <path>`. See the `blast --help` output (generated from
`cmd/root.go`'s `Long` description) for the exact precedence rules. This
logic intentionally lives in `cmd/root.go` only — no other package needs
to know the alias exists.

## What's not implemented yet

- `blast diff`, `blast graph`, `blast history` (stubs only / not yet
  wired to a command).
- Git analyzer (`internal/git`) — churn, co-change frequency, history
  window.
- CI analyzer (`internal/ci`) — GitHub Actions workflow relevance.
- Risk engine (`internal/risk`) — explainable scoring, critical-path
  keyword weighting.
- `.changeblast.yml` configuration.

These are scaffolded as empty packages/directories per the target
architecture so later work doesn't require a restructuring, but contain
no logic yet.
