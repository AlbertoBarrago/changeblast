# ChangeBlast — Project Notes

## VCS: Jujutsu (jj), not plain git
This repo is colocated with git but the working VCS is **jj**. Use `jj`
commands for the day-to-day flow, not `git commit`/`git checkout`:

- `jj status`, `jj log` to inspect state (working-copy changes are
  already a commit under `@`, no separate staging step)
- `jj describe -m "..."` to set/update the message on the current change
- `jj new` to start a new change on top of the current one
- `jj git push` to push to the colocated git remote
- Plain `git` commands (status/log/diff) still work for read-only
  inspection since the repo is colocated, but don't use `git commit`,
  `git add`, or `git checkout` for the actual workflow.
