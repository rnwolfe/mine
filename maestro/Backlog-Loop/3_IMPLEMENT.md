# Implement Feature

## Context

- **Playbook:** Backlog Loop
- **Agent:** Mine CLI
- **Project:** /home/rnwolfe/dev/mine
- **Auto Run Folder:** /home/rnwolfe/dev/mine/maestro
- **Loop:** 00001

## Objective

Execute the implementation plan from `/home/rnwolfe/dev/mine/maestro/LOOP_00001_PLAN.md`. Write code, write tests, and verify everything builds and passes. All work happens in the worktree.

## Tasks

- [x] **Read the plan and locate worktree**: Plan at `LOOP_00001_PLAN.md`, worktree at `/home/rnwolfe/dev/mine-worktrees/issue-19`.

- [x] **Implement the feature**: Created `internal/analytics/` package (analytics.go, id.go) with fire-and-forget Ping(), daily dedup via kv table, installation ID persistence. Added `AnalyticsConfig` to config with `*bool` pointer semantics (nil = enabled). Added `config set/get` subcommands. Integrated analytics ping via `PersistentPostRun` on rootCmd. Added one-time privacy notice. Added privacy docs page to site. Generated analytics ID during `mine init`.

- [x] **Write tests**: Created analytics_test.go (9 tests) and id_test.go (5 tests) covering: payload construction, no-extra-fields audit, opt-out behavior, HTTP send verification, daily dedup, dedup reset across days, network failure silence, ShowNotice first-time/only-once, UUID generation/persistence/corruption/empty-file, UUID validation.

- [x] **Run tests**: All 17 packages pass with race detector enabled. 14 new analytics tests + all existing tests green.

- [x] **Run build**: Binary builds successfully with ldflags.

- [x] **Verify protected files**: `git diff --name-only` shows only: cmd/config.go, cmd/init.go, cmd/root.go, go.mod, internal/config/config.go, site/src/config/sidebar.json. No protected files modified.

## Guidelines

- Implement exactly what the plan specifies — don't add unplanned features
- If you discover the plan has a flaw, make a reasonable adjustment and note it
- Run `make test` frequently during implementation, not just at the end
- Prefer editing existing files over creating new ones where it makes sense
- Don't add unnecessary abstractions — keep it simple
- All file paths are relative to the worktree directory, NOT the main project root
