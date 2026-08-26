# ChangeBlast: Manual Test Guide

Guide for a live manual testing session, with `blast` installed via Homebrew.

## 0. Setup

```bash
brew update
brew upgrade changeblast   # or: brew install changeblast if not installed yet
blast version
blast doctor
```

**Important:** always check `blast version` before starting. If it shows an
older version than expected, run `brew update && brew upgrade changeblast`
first.

## 1. Test repositories

Real repositories for testing at different scales and languages (in the
scratchpad, not in this repo):

| Repo | Language | Files | Notes |
|---|---|---|---|
| `sindresorhus/got` | JS/TS | ~85 | small/medium HTTP library, has tsconfig |
| `date-fns/date-fns` | JS/TS | ~1600 | large library, good scale test |
| ChangeBlast itself (`/Users/albz/Projects/Blast`) | Go | ~50 | direct dogfooding, already has go.mod |

```bash
cd <scratchpad>/bench-got     # or bench-datefns
blast doctor
```

For the Go test, just stay in the project's own directory:
```bash
cd /Users/albz/Projects/Blast
blast inspect internal/graph/graph.go
```

## 2. Core commands: checklist

- [ ] `blast inspect <path>` on a file with known direct/indirect dependents
- [ ] `blast inspect <path> --json`: validate with `jq .` that the JSON is well-formed
- [ ] `blast <path>` (alias): must produce the same output as `blast inspect <path>`
- [ ] `blast <path> --json` (alias with flag) → must work exactly like `blast inspect <path> --json`
- [ ] `blast diff` (no changes) → "No JS/TS module changes found"
- [ ] Modify a file, `blast diff` → shows the impact of the modified file
- [ ] `blast diff HEAD~1` (or an older ref) → shows impact over a wider range
- [ ] `blast graph <path>` → direct dependencies + direct dependents
- [ ] `blast history <path>` → churn and co-change consistent with `git log`
- [ ] `blast doctor` in a non-git directory → should fail with a clear message
- [ ] `blast inspect <nonexistent path>` → clear error, exit code 1
- [ ] `blast inspect <path> --fail-on high` on a low-risk file → exit 0
- [ ] `blast inspect <path> --fail-on low` → should almost always exit with code 2

Check the exit code with:
```bash
blast inspect <path> --fail-on low; echo "exit: $?"
```

### Optional path defaults (`blast inspect`/`blast history` with no argument)

- [ ] `blast inspect` (no arguments) → equivalent to `blast inspect .`
- [ ] `blast history` (no arguments) → equivalent to `blast history .`, must not error

### Directory analysis (`blast inspect <dir>`)

- [ ] `blast inspect .` at the project root → risk-sorted summary, with a final HIGH/MEDIUM/LOW count
- [ ] `blast inspect <subfolder>` → only files inside that folder, not the whole repo
- [ ] `blast inspect . --json` → JSON array, same shape as `blast diff --json`
- [ ] `blast inspect . --fail-on high` → gates on the worst-risk file found

### Output to file (`--output`/`-o`)

- [ ] `blast inspect <path> --output report.txt` → the file contains the same output that would go to stdout, with no ANSI color codes even when run from a terminal
- [ ] `blast inspect <path> --output report.json --json` → valid JSON in the file
- [ ] `blast diff --output diff-report.txt`, `blast graph <path> -o graph.txt`, `blast history <path> -o hist.txt` → same behavior

### Go support

- [ ] `blast inspect internal/graph/graph.go` (from the ChangeBlast repo) → correctly shows the files that import that package
- [ ] `blast doctor` in a Go repo with no `go.mod` → Go imports left unresolved (no error, just no resolved dependencies)
- [ ] Standard library imports (`fmt`, `os`, etc.) → produce no edge in the graph (correctly treated as external)
- [ ] Importing an external module (e.g. `github.com/spf13/cobra`) → not traversed, treated as external

### AI explanation (`--explain`, requires local Ollama)

- [ ] `ollama serve` running and at least one model pulled (`ollama pull llama3.2` or any other)
- [ ] `blast doctor` → an "Ollama" line with reachable/not-reachable status matching whether `ollama serve` is running, and (when reachable) the actual list of pulled model names
- [ ] `blast inspect <file> --explain` → after the deterministic report, an "Explanation (ollama)" section with text that references the actual signals (not generic filler)
- [ ] The explanation text must be plain prose: no literal `**asterisks**`, backticks, or `###` headers left in the output
- [ ] `blast inspect <file> --explain --explain-model <other-model>` → uses the specified model
- [ ] With no `--explain-model` and the default model (`llama3.2`) not pulled → blast should automatically fall back to a model you do have (check `blast doctor`'s model list), not fail outright
- [ ] `blast inspect <file> --explain --json` → JSON shaped as `{"analysis": {...}, "explanation": "..."}` instead of the flat shape
- [ ] `blast inspect <file> --json` (no `--explain`) → must stay in the usual flat shape (no `analysis` field), confirming `--explain` never changes default `--json` output
- [ ] With Ollama stopped, `blast inspect <file> --explain` → the deterministic report still appears, with "unavailable: ..." instead of the explanation, exit code unchanged
- [ ] `blast inspect <file> --explain --explain-host http://127.0.0.1:1` (unreachable host) → same graceful "unavailable" behavior, no crash
- [ ] Confirm no network call happens at all without `--explain` (e.g. point `--explain-host` at an unreachable address on a plain `--json` run with no `--explain`: must succeed normally)

## 3. Colors and terminal

```bash
blast inspect <path>                          # colored if the terminal is a TTY
blast inspect <path> | cat                    # no color (piped output)
NO_COLOR=1 blast inspect <path>               # no color even in a TTY
```

## 4. Performance benchmarks

Measure timing on both repositories:

```bash
cd <scratchpad>/bench-got
time blast inspect source/index.ts
time blast diff HEAD~5

cd <scratchpad>/bench-datefns
time blast inspect src/index.ts
time blast diff HEAD~5
```

What to watch for:
- Total scan time (dominated by `filepath.WalkDir` + regex import matching per file)
- Whether `blast diff` with multiple changed files stays close to a single scan's time (single shared scan, reused per file)
- Any unexpected slowdown on repositories where `node_modules` isn't properly excluded

For a more precise comparison, repeat 3 times and take the best time (the first run warms the filesystem cache):

```bash
for i in 1 2 3; do time blast inspect src/index.ts; done
```

## 5. Homebrew: package verification

```bash
brew info changeblast
brew test changeblast     # runs the formula's test (blast version)
brew audit --strict changeblast   # optional, checks formula quality
man blast                 # should render, not "No manual entry"
man blast-inspect          # subcommand man pages should also work
```

## 6. What to report

For every issue found, note: the exact command run, the output you got, the
output you expected, the version (`blast version`), and whether it
reproduces on both `got` and `date-fns` or only one of them.
