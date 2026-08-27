# Serval: Manual Test Guide

Guide for a live manual testing session, with `serval` installed via Homebrew.

## 0. Setup

```bash
brew update
brew upgrade AlbertoBarrago/tap/serval   # or: brew install AlbertoBarrago/tap/serval if not installed yet
serval version
serval doctor
```

Check `serval version` before starting. If it shows an older
version than expected, run `brew update && brew upgrade
AlbertoBarrago/tap/serval` first.

## 1. Test repositories

Real repositories for testing at different scales and languages (in the
scratchpad, not in this repo):

| Repo | Language | Files | Notes |
|---|---|---|---|
| `sindresorhus/got` | JS/TS | ~85 | small/medium HTTP library, has tsconfig |
| `date-fns/date-fns` | JS/TS | ~1600 | large library, good scale test |
| Serval itself (`/Users/albz/Projects/Blast`) | Go | ~50 | direct dogfooding, already has go.mod |

```bash
cd <scratchpad>/bench-got     # or bench-datefns
serval doctor
```

For the Go test, just stay in the project's own directory:
```bash
cd /Users/albz/Projects/Blast
serval inspect internal/graph/graph.go
```

## 2. Core commands: checklist

- [x] `serval inspect <path>` on a file with known direct/indirect dependents — **BUG**: fails on `.js`-extension imports (see report)
- [x] `serval inspect <path> --json`: validate with `jq .` that the JSON is well-formed — OK
- [x] `serval <path>` (alias): must produce the same output as `serval inspect <path>` — content matches, but see non-determinism bug in report
- [x] `serval <path> --json` (alias with flag) → must work exactly like `serval inspect <path> --json` — OK
- [x] `serval diff` (no changes) → "No JS/TS module changes found" — OK (actual message: "No recognized-module changes found against HEAD")
- [x] Modify a file, `serval diff` → shows the impact of the modified file — OK
- [x] `serval diff HEAD~1` (or an older ref) → shows impact over a wider range — OK
- [x] `serval graph <path>` → direct dependencies + direct dependents — OK
- [x] `serval history <path>` → churn and co-change consistent with `git log` — OK
- [x] `serval doctor` in a non-git directory → should fail with a clear message — OK, exit 1
- [x] `serval inspect <nonexistent path>` → clear error, exit code 1 — OK
- [x] `serval inspect <path> --fail-on high` on a low-risk file → exit 0 — OK
- [x] `serval inspect <path> --fail-on low` → should almost always exit with code 2 — OK

Check the exit code with:
```bash
serval inspect <path> --fail-on low; echo "exit: $?"
```

### Optional path defaults (`serval inspect`/`serval history` with no argument)

- [x] `serval inspect` (no arguments) → equivalent to `serval inspect .` — OK
- [x] `serval history` (no arguments) → equivalent to `serval history .`, must not error — OK

### Directory analysis (`serval inspect <dir>`)

- [x] `serval inspect .` at the project root → risk-sorted summary, with a final HIGH/MEDIUM/LOW count — OK
- [x] `serval inspect <subfolder>` → only files inside that folder, not the whole repo — OK
- [x] `serval inspect . --json` → JSON array, same shape as `serval diff --json` — OK
- [x] `serval inspect . --fail-on high` → gates on the worst-risk file found — OK

### Output to file (`--output`/`-o`)

- [x] `serval inspect <path> --output report.txt` → the file contains the same output that would go to stdout, with no ANSI color codes even when run from a terminal — OK
- [x] `serval inspect <path> --output report.json --json` → valid JSON in the file — OK
- [x] `serval diff --output diff-report.txt`, `serval graph <path> -o graph.txt`, `serval history <path> -o hist.txt` → same behavior — OK

### Go support

- [x] `serval inspect internal/graph/graph.go` (from the Serval repo) → correctly shows the files that import that package — OK
- [ ] `serval doctor` in a Go repo with no `go.mod` → Go imports left unresolved (no error, just no resolved dependencies) — not tested
- [x] Standard library imports (`fmt`, `os`, etc.) → produce no edge in the graph (correctly treated as external) — OK, confirmed via graph output
- [ ] Importing an external module (e.g. `github.com/spf13/cobra`) → not traversed, treated as external — not explicitly isolated, but consistent with observed behavior

### Other languages (Python, Java, C)

Not in the original guide's repo list, but serval ships resolvers for these
(`pyresolve`, `javaresolve`, `cresolve` internally), so added ad-hoc minimal
fixture repos in the scratchpad to cover them:

- [x] Python: `from pkg.core import helper`, `from . import core`, `from
  .core import helper` all resolve direct dependents correctly — OK
- [x] Java: `import com.example.core.Helper;` resolves direct dependents
  correctly — OK
- [x] C: `#include "../include/core.h"` resolves direct dependents
  correctly; `#include <stdio.h>` (system header) correctly produces no
  edge — OK
- [x] Non-determinism of impact-list ordering (see bug report) reproduces
  on Python (visible with just 3 files) and Java (1/5 runs), confirming
  it's a general bug, not JS/TS-specific

### AI explanation (`--explain`, requires local Ollama)

- [x] `ollama serve` running and at least one model pulled (`ollama pull llama3.2` or any other) — 4 models pulled
- [x] `serval doctor` → an "Ollama" line with reachable/not-reachable status matching whether `ollama serve` is running, and (when reachable) the actual list of pulled model names — OK
- [x] `serval inspect <file> --explain` → after the deterministic report, an "Explanation (ollama)" section with text that references the actual signals (not generic filler) — OK, references real signals
- [x] The explanation text must be plain prose: no literal `**asterisks**`, backticks, or `###` headers left in the output — OK
- [ ] `serval inspect <file> --explain --explain-model <other-model>` → uses the specified model — not tested
- [ ] With no `--explain-model` and the default model (`llama3.2`) not pulled → serval should automatically fall back to a model you do have (check `serval doctor`'s model list), not fail outright — not tested
- [x] `serval inspect <file> --explain --json` → JSON shaped as `{"analysis": {...}, "explanation": "..."}` instead of the flat shape — OK
- [x] `serval inspect <file> --json` (no `--explain`) → must stay in the usual flat shape (no `analysis` field), confirming `--explain` never changes default `--json` output — OK
- [ ] With Ollama stopped, `serval inspect <file> --explain` → the deterministic report still appears, with "unavailable: ..." instead of the explanation, exit code unchanged — not tested directly, but unreachable-host case below confirms the graceful path
- [x] `serval inspect <file> --explain --explain-host http://127.0.0.1:1` (unreachable host) → same graceful "unavailable" behavior, no crash — OK
- [x] Confirm no network call happens at all without `--explain` (e.g. point `--explain-host` at an unreachable address on a plain `--json` run with no `--explain`: must succeed normally) — OK, ~33ms, no hang

## 3. Colors and terminal

- [x] piped output has no ANSI codes — OK
- [x] `NO_COLOR=1` has no ANSI codes — OK

```bash
serval inspect <path>                          # colored if the terminal is a TTY
serval inspect <path> | cat                    # no color (piped output)
NO_COLOR=1 serval inspect <path>               # no color even in a TTY
```

## 4. Performance benchmarks

- [x] `bench-got`: `serval inspect source/index.ts` ~0.05s (best of 3) — excellent

Measure timing on both repositories:

```bash
cd <scratchpad>/bench-got
time serval inspect source/index.ts
time serval diff HEAD~5

cd <scratchpad>/bench-datefns
time serval inspect src/index.ts
time serval diff HEAD~5
```

What to watch for:
- Total scan time (dominated by `filepath.WalkDir` + regex import matching per file)
- Whether `serval diff` with multiple changed files stays close to a single scan's time (single shared scan, reused per file)
- Any unexpected slowdown on repositories where `node_modules` isn't properly excluded

For a more precise comparison, repeat 3 times and take the best time (the first run warms the filesystem cache):

```bash
for i in 1 2 3; do time serval inspect src/index.ts; done
```

## 5. Homebrew: package verification

- [x] `brew test AlbertoBarrago/tap/serval` — OK
- [x] `man serval` — renders correctly
- [x] `man serval-inspect` — renders correctly
- [ ] `brew audit --strict` — not run (optional)

```bash
brew info AlbertoBarrago/tap/serval
brew test AlbertoBarrago/tap/serval     # runs the formula's test (serval version)
brew audit --strict AlbertoBarrago/tap/serval   # optional, checks formula quality
man serval                 # should render, not "No manual entry"
man serval-inspect          # subcommand man pages should also work
```

## 6. What to report

For every issue found, note: the exact command run, the output you got, the
output you expected, the version (`serval version`), and whether it
reproduces on both `got` and `date-fns` or only one of them.

## 7. Bugs found (2026-08-27, serval 0.1.18)

Both bugs below have since been **fixed** in this working tree (not yet
released as a new version). Fixes and regression tests:

- `.js`→`.ts` resolution: `internal/repository/resolve.go` — `resolveOnDisk`
  now strips a trailing NodeNext output extension (`.js`, `.jsx`, `.mjs`,
  `.cjs`) and retries resolution on the base path when no literal file
  exists, instead of only ever appending candidate extensions. Also added
  `.mts`/`.cts` to `candidateExtensions` for completeness. Regression tests:
  `TestResolver_NodeNextJSExtensionResolvesToTypeScriptSource`,
  `TestResolver_LiteralJSFileTakesPrecedenceOverTypeScriptRewrite`.
- Non-deterministic ordering: `internal/graph/graph.go` — `keys()` (backing
  `Dependents`, `Dependencies`, `Nodes`) now sorts before returning, fixing
  the shared root cause across every resolver/language, not just JS/TS.
  Regression tests: `TestDependents_ReturnedInSortedOrder`,
  `TestNodes_ReturnedInSortedOrder`.

Verified post-fix: `got`'s `source/core/index.ts` now correctly shows
`source/index.ts`, `source/create.ts`, `source/types.ts`, etc. as direct
dependents, and 10/10 repeated `inspect` runs on both `got` and the Python
fixture produced byte-identical output.

### HIGH — `.js`-extension imports never resolve (TS/ESM NodeNext pattern)

- **Command**: `serval inspect source/core/index.ts` (in `bench-got`)
- **Got**: `Direct impact (none found)` / `Indirect impact (none found)`
- **Expected**: `source/index.ts`, `source/create.ts`, `source/types.ts`,
  `source/as-promise/index.ts`, `source/as-promise/types.ts` as direct
  dependents (all import via `from './core/index.js'`)
- **Root cause**: `got` uses the standard TS `NodeNext`/ESM pattern where
  relative imports use a `.js` suffix that actually resolves to a `.ts`
  source file. All 76 relative imports in `got` use this pattern, and
  **zero** are resolved by serval, i.e. direct/indirect impact is always
  empty for the entire repo.
- **Reproduces on**: `got` only (confirmed). `date-fns` uses explicit `.ts`
  extensions in imports and resolves correctly, so this is specific to the
  `.js`→`.ts` resolution step, not resolution in general.
- **Impact**: this pattern is extremely common in modern TS/npm packages
  (any package targeting `"module": "NodeNext"`), so this likely breaks
  `inspect`/`diff`/`graph` direct-impact detection on a large fraction of
  real-world TypeScript codebases.

### MEDIUM — non-deterministic ordering of direct/indirect impact lists

- **Command**: run `serval inspect pkgs/core/src/constants/index.ts` twice
  in a row (in `bench-datefns`)
- **Got**: the `Direct impact` list has a different order each run (same
  set of files, shuffled)
- **Expected**: stable, deterministic ordering (e.g. sorted by path) across
  repeated runs of the same command against the same repo state
- **Reproduces on**: `date-fns` (confirmed), and also confirmed with
  minimal ad-hoc fixture repos in **Python** (visible with just 3 files)
  and **Java** (1/5 runs). Not JS/TS-specific: this is a general bug
  across every language resolver, consistent with unsorted Go map
  iteration somewhere in the shared impact-aggregation code path.
- **Impact**: makes `--output`/diff-based workflows and any CI gating on
  exact output unreliable; also makes `serval <path>` vs `serval inspect
  <path>` look inconsistent even though it's the same underlying bug, not
  an alias problem.

### LOW — `diff` "no changes" message text differs from TESTING.md expectation

- **Got**: `No recognized-module changes found against HEAD.`
- **TESTING.md expected**: `No JS/TS module changes found`
- Purely a documentation/wording mismatch, not a functional bug (Go
  support was added since the guide was written). Worth updating
  TESTING.md's wording, not the tool.

## 8. Suggested remediation, by severity

1. **[HIGH] — Fixed.** `internal/repository/resolve.go`: `resolveOnDisk`
   now strips a trailing NodeNext output extension (`.js`, `.jsx`, `.mjs`,
   `.cjs`) and retries on the base path (trying `.ts`, `.tsx`, `.mts`,
   `.cts`, etc.) when no literal file exists, instead of only ever
   appending candidate extensions to the full specifier. Covered by
   `TestResolver_NodeNextJSExtensionResolvesToTypeScriptSource` and
   `TestResolver_LiteralJSFileTakesPrecedenceOverTypeScriptRewrite`.
2. **[MEDIUM] — Fixed.** `internal/graph/graph.go`: `keys()` (the shared
   helper backing `Dependents`, `Dependencies`, and `Nodes`) now sorts its
   output. This was the single shared root cause across every
   language/resolver (JS/TS, Python, Java, C, Go all go through the same
   graph), so one fix covers all of them. `history`'s co-changed list was
   checked separately and was already stable (tie-broken by path).
   Covered by `TestDependents_ReturnedInSortedOrder` and
   `TestNodes_ReturnedInSortedOrder`.
3. **[LOW]** Update `TESTING.md`'s expected message for the no-op `diff`
   case to match current wording, or make the wording configurable/stable
   across doc and code (low priority, doc-only, not addressed here).
