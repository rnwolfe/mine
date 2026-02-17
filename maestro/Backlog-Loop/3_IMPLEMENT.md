# Implement Feature

## Context

- **Playbook:** Backlog Loop
- **Agent:** Mine CLI
- **Project:** /home/rnwolfe/dev/mine
- **Auto Run Folder:** /home/rnwolfe/dev/mine/maestro
- **Loop:** 00002

## Objective

Execute the implementation plan from `/home/rnwolfe/dev/mine/maestro/LOOP_00002_PLAN.md`. Write code, write tests, and verify everything builds and passes. All work happens in the worktree.

## Tasks

- [x] **Read the plan and locate worktree**: Plan at `LOOP_00002_PLAN.md`, worktree at `/home/rnwolfe/dev/mine-worktrees/issue-28`. Worktree exists on branch `maestro/issue-28-user-local-hooks`. Plan confirms all functional code is already implemented — remaining work is documentation only.

- [x] **Implement the feature**: All functional hook code was pre-existing (discover.go, exec.go, hook.go, pipeline.go, registry.go, cmd/hook.go). Created documentation: `site/src/content/docs/commands/hook.md` (user-facing command reference with filename conventions, stages, modes, JSON protocol, and CLI usage). Created 3 example hook scripts in `docs/examples/hooks/`: `todo.add.preexec.sh` (auto-tag transform hook), `todo.done.notify.sh` (completion logging notify hook), `all-commands.notify.py` (Python command logger). Updated `CLAUDE.md` with architecture pattern #11 (user-local hooks), added `registry.go` and `cmd/hook.go` to Key Files table.

- [x] **Write tests**: No new tests needed — documentation-only changes. Existing 524 lines of tests across `hook_test.go` (353 lines) and `discover_test.go` (171 lines) comprehensively cover all acceptance criteria. Example scripts validated via `bash -n` and `python3 -c py_compile`.

- [x] **Run tests**: All 15 packages pass with race detector (`go test ./... -count=1 -race`). All existing tests green.

- [x] **Run build**: Binary builds successfully with `-ldflags="-s -w"`.

- [x] **Verify protected files**: `git diff --name-only` shows only: CLAUDE.md (planned update). New untracked files: `site/src/content/docs/commands/hook.md`, `docs/examples/hooks/` (3 scripts). No workflows, autodev scripts, or other protected files modified.

## Guidelines

- Implement exactly what the plan specifies — don't add unplanned features
- If you discover the plan has a flaw, make a reasonable adjustment and note it
- Run `make test` frequently during implementation, not just at the end
- Prefer editing existing files over creating new ones where it makes sense
- Don't add unnecessary abstractions — keep it simple
- All file paths are relative to the worktree directory, NOT the main project root
