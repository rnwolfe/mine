# Plan Implementation

## Context

- **Playbook:** Backlog Loop
- **Agent:** Mine CLI
- **Project:** /home/rnwolfe/dev/mine
- **Auto Run Folder:** /home/rnwolfe/dev/mine/maestro
- **Loop:** 00001

## Objective

Read the selected issue from `/home/rnwolfe/dev/mine/maestro/LOOP_00001_ISSUE.md`, explore the codebase in the worktree, and produce a concrete implementation plan. This is the thinking phase — no code changes yet.

## Tasks

- [x] **Read issue and locate worktree**: Read `/home/rnwolfe/dev/mine/maestro/LOOP_00001_ISSUE.md` for the issue details and worktree path. If the status is not `READY`, mark this task complete without proceeding. Extract the worktree path from the `## Worktree` section — all codebase exploration happens in that directory.
  - Status: READY. Worktree at `/home/rnwolfe/dev/mine-worktrees/issue-19`. Issue #19: Anonymous usage analytics (opt-out).

- [x] **Read project conventions**: Read `CLAUDE.md` (from the worktree or project root) for project conventions, architecture patterns, and file organization rules.
  - Reviewed both main and worktree CLAUDE.md. Key patterns: domain separation (internal/), store pattern (SQLite + kv table), UI helpers, hook.Wrap pipeline, XDG paths, < 500 lines per file.

- [x] **Explore relevant code**: Based on the issue requirements, explore the codebase **in the worktree directory** to understand:
  - Which existing packages/files are relevant
  - What patterns are already used (command structure, domain packages, store layer, UI helpers)
  - Where new code should live following existing conventions
  - What existing tests look like for similar functionality
  - Explored: `cmd/root.go` (command registration, hook.Wrap, Execute), `internal/config/config.go` (TOML config, XDG paths, Load/Save), `internal/store/store.go` (SQLite, kv table, migrations), `internal/hook/pipeline.go` (4-stage pipeline, notify = fire-and-forget goroutine), `cmd/init.go` (first-run flow, AI setup), `cmd/config.go` (config display, no `set` subcommand yet), `internal/meta/meta.go` (SystemInfo with OS/arch/version), `internal/todo/todo_test.go` (test pattern: in-memory SQLite, table setup), `internal/version/version.go` (build-time version), `internal/ui/` (theme + print helpers). Found: `google/uuid` already an indirect dep, kv table used by dig for state, no `config set` command exists.

- [x] **Design implementation approach**: Write a plan to `/home/rnwolfe/dev/mine/maestro/LOOP_00001_PLAN.md` with:

```markdown
# Implementation Plan: Issue #N — Title

## Approach
<2-3 sentences describing the overall approach>

## Files to Create
- `path/to/new/file.go` — purpose
- `path/to/new/file_test.go` — what it tests

## Files to Modify
- `path/to/existing.go` — what changes and why

## Architecture Decisions
- <Any design choices: data model, package boundaries, API surface>

## CLI Surface
- `mine <command> [flags]` — description of new commands/subcommands
- New flags on existing commands (if any)

## Test Strategy
- Unit tests for: <list>
- Edge cases: <list>
- Integration points: <list>

## Risks & Considerations
- <Anything tricky, any existing code that might conflict>

## Acceptance Criteria Mapping
- [ ] Criterion from issue -> planned implementation detail
- [ ] ...
```

## Guidelines

- Follow the project's domain separation pattern: thin `cmd/` orchestration, domain logic in `internal/<package>/`
- Keep files under 500 lines
- Use the store pattern (`store.DB` wrapper) for any new data persistence
- All output through `internal/ui` helpers — never raw `fmt.Println`
- Do NOT plan changes to CLAUDE.md, `.github/workflows/`, or `scripts/autodev/`
