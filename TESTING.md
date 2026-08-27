# Impactline: Manual Test Guide

Guide for a live manual testing session, with `impactline` installed via Homebrew.

## 0. Setup

```bash
brew update
brew upgrade AlbertoBarrago/tap/impactline   # or: brew install AlbertoBarrago/tap/impactline if not installed yet
impactline version
impactline doctor
```

Check `impactline version` before starting. If it shows an older
version than expected, run `brew update && brew upgrade
AlbertoBarrago/tap/impactline` first.

## 1. Test repositories

Real repositories for testing at different scales and languages (in the
scratchpad, not in this repo):

| Repo | Language | Files | Notes |
|---|---|---|---|
| `sindresorhus/got` | JS/TS | ~85 | small/medium HTTP library, has tsconfig |
| `date-fns/date-fns` | JS/TS | ~1600 | large library, good scale test |
| Impactline itself (`/Users/albz/Projects/Blast`) | Go | ~50 | direct dogfooding, already has go.mod |

```bash
cd <scratchpad>/bench-got     # or bench-datefns
impactline doctor
```

For the Go test, just stay in the project's own directory:
```bash
cd /Users/albz/Projects/Blast
impactline inspect internal/graph/graph.go
```

## 2. Core commands: checklist

- [ ] `impactline inspect <path>` on a file with known direct/indirect dependents
- [ ] `impactline inspect <path> --json`: validate with `jq .` that the JSON is well-formed
- [ ] `impactline <path>` (alias): must produce the same output as `impactline inspect <path>`
- [ ] `impactline <path> --json` (alias with flag) → must work exactly like `impactline inspect <path> --json`
- [ ] `impactline diff` (no changes) → "No JS/TS module changes found"
- [ ] Modify a file, `impactline diff` → shows the impact of the modified file
- [ ] `impactline diff HEAD~1` (or an older ref) → shows impact over a wider range
- [ ] `impactline graph <path>` → direct dependencies + direct dependents
- [ ] `impactline history <path>` → churn and co-change consistent with `git log`
- [ ] `impactline doctor` in a non-git directory → should fail with a clear message
- [ ] `impactline inspect <nonexistent path>` → clear error, exit code 1
- [ ] `impactline inspect <path> --fail-on high` on a low-risk file → exit 0
- [ ] `impactline inspect <path> --fail-on low` → should almost always exit with code 2

Check the exit code with:
```bash
impactline inspect <path> --fail-on low; echo "exit: $?"
```

### Optional path defaults (`impactline inspect`/`impactline history` with no argument)

- [ ] `impactline inspect` (no arguments) → equivalent to `impactline inspect .`
- [ ] `impactline history` (no arguments) → equivalent to `impactline history .`, must not error

### Directory analysis (`impactline inspect <dir>`)

- [ ] `impactline inspect .` at the project root → risk-sorted summary, with a final HIGH/MEDIUM/LOW count
- [ ] `impactline inspect <subfolder>` → only files inside that folder, not the whole repo
- [ ] `impactline inspect . --json` → JSON array, same shape as `impactline diff --json`
- [ ] `impactline inspect . --fail-on high` → gates on the worst-risk file found

### Output to file (`--output`/`-o`)

- [ ] `impactline inspect <path> --output report.txt` → the file contains the same output that would go to stdout, with no ANSI color codes even when run from a terminal
- [ ] `impactline inspect <path> --output report.json --json` → valid JSON in the file
- [ ] `impactline diff --output diff-report.txt`, `impactline graph <path> -o graph.txt`, `impactline history <path> -o hist.txt` → same behavior

### Go support

- [ ] `impactline inspect internal/graph/graph.go` (from the Impactline repo) → correctly shows the files that import that package
- [ ] `impactline doctor` in a Go repo with no `go.mod` → Go imports left unresolved (no error, just no resolved dependencies)
- [ ] Standard library imports (`fmt`, `os`, etc.) → produce no edge in the graph (correctly treated as external)
- [ ] Importing an external module (e.g. `github.com/spf13/cobra`) → not traversed, treated as external

### AI explanation (`--explain`, requires local Ollama)

- [ ] `ollama serve` running and at least one model pulled (`ollama pull llama3.2` or any other)
- [ ] `impactline doctor` → an "Ollama" line with reachable/not-reachable status matching whether `ollama serve` is running, and (when reachable) the actual list of pulled model names
- [ ] `impactline inspect <file> --explain` → after the deterministic report, an "Explanation (ollama)" section with text that references the actual signals (not generic filler)
- [ ] The explanation text must be plain prose: no literal `**asterisks**`, backticks, or `###` headers left in the output
- [ ] `impactline inspect <file> --explain --explain-model <other-model>` → uses the specified model
- [ ] With no `--explain-model` and the default model (`llama3.2`) not pulled → impactline should automatically fall back to a model you do have (check `impactline doctor`'s model list), not fail outright
- [ ] `impactline inspect <file> --explain --json` → JSON shaped as `{"analysis": {...}, "explanation": "..."}` instead of the flat shape
- [ ] `impactline inspect <file> --json` (no `--explain`) → must stay in the usual flat shape (no `analysis` field), confirming `--explain` never changes default `--json` output
- [ ] With Ollama stopped, `impactline inspect <file> --explain` → the deterministic report still appears, with "unavailable: ..." instead of the explanation, exit code unchanged
- [ ] `impactline inspect <file> --explain --explain-host http://127.0.0.1:1` (unreachable host) → same graceful "unavailable" behavior, no crash
- [ ] Confirm no network call happens at all without `--explain` (e.g. point `--explain-host` at an unreachable address on a plain `--json` run with no `--explain`: must succeed normally)

## 3. Colors and terminal

```bash
impactline inspect <path>                          # colored if the terminal is a TTY
impactline inspect <path> | cat                    # no color (piped output)
NO_COLOR=1 impactline inspect <path>               # no color even in a TTY
```

## 4. Performance benchmarks

Measure timing on both repositories:

```bash
cd <scratchpad>/bench-got
time impactline inspect source/index.ts
time impactline diff HEAD~5

cd <scratchpad>/bench-datefns
time impactline inspect src/index.ts
time impactline diff HEAD~5
```

What to watch for:
- Total scan time (dominated by `filepath.WalkDir` + regex import matching per file)
- Whether `impactline diff` with multiple changed files stays close to a single scan's time (single shared scan, reused per file)
- Any unexpected slowdown on repositories where `node_modules` isn't properly excluded

For a more precise comparison, repeat 3 times and take the best time (the first run warms the filesystem cache):

```bash
for i in 1 2 3; do time impactline inspect src/index.ts; done
```

## 5. Homebrew: package verification

```bash
brew info AlbertoBarrago/tap/impactline
brew test AlbertoBarrago/tap/impactline     # runs the formula's test (impactline version)
brew audit --strict AlbertoBarrago/tap/impactline   # optional, checks formula quality
man impactline                 # should render, not "No manual entry"
man impactline-inspect          # subcommand man pages should also work
```

## 6. What to report

For every issue found, note: the exact command run, the output you got, the
output you expected, the version (`impactline version`), and whether it
reproduces on both `got` and `date-fns` or only one of them.
