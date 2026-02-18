# Plan Implementation

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Read the selected issue from `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md`, explore the codebase in the worktree, and produce a concrete implementation plan. This is the thinking phase — no code changes yet.

## Tasks

- [ ] **Read issue and locate worktree**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` for the issue details and worktree path. If the status is not `READY`, mark this task complete without proceeding. Extract the worktree path from the `## Worktree` section — all codebase exploration happens in that directory.

- [ ] **Read project conventions**: Read `CLAUDE.md` (from the worktree or project root) for project conventions, architecture patterns, and file organization rules.

- [ ] **Explore relevant code**: Based on the issue requirements, explore the codebase **in the worktree directory** to understand:
  - Which existing packages/files are relevant
  - What patterns are already used (command structure, domain packages, store layer, UI helpers)
  - Where new code should live following existing conventions
  - What existing tests look like for similar functionality

- [ ] **Design implementation approach**: Write a plan to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md` with:

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
