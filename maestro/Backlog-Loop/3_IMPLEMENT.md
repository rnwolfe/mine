# Implement Feature

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Execute the implementation plan from `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md`. Write code, write tests, and verify everything builds and passes. All work happens in the worktree.

## Tasks

- [ ] **Read the plan and locate worktree**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md`. If the file doesn't exist, mark this task complete without proceeding — there's nothing to implement. Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` to get the worktree path from the `## Worktree` section. All code changes happen in this directory.

- [ ] **Implement the feature**: Following the plan, create and modify files **in the worktree directory**. Adhere to these rules:
  - Read CLAUDE.md for project conventions
  - Follow existing code patterns and style
  - Do NOT modify CLAUDE.md, any files in `.github/workflows/`, or `scripts/autodev/`
  - Keep files under 500 lines
  - Use `internal/ui` helpers for all output
  - Use the store pattern for data persistence

- [ ] **Write tests**: Create `_test.go` files alongside the code in the worktree. Cover:
  - Happy path for each new function
  - Error cases and edge cases
  - Any acceptance criteria that can be verified programmatically

- [ ] **Run tests**: Execute `make test` **from the worktree directory** (e.g., `make -C WORKTREE_PATH test` or `cd WORKTREE_PATH && make test`) and verify all tests pass (including existing tests). If tests fail, fix the issues before proceeding.

- [ ] **Run build**: Execute `make build` **from the worktree directory** and verify the binary builds successfully. If the build fails, fix the issues.

- [ ] **Verify protected files**: Run `git -C WORKTREE_PATH diff --name-only` and confirm no changes to CLAUDE.md, `.github/workflows/`, or `scripts/autodev/`. If any protected files were modified, revert them with `git -C WORKTREE_PATH checkout -- <file>`.

## Guidelines

- Implement exactly what the plan specifies — don't add unplanned features
- If you discover the plan has a flaw, make a reasonable adjustment and note it
- Run `make test` frequently during implementation, not just at the end
- Prefer editing existing files over creating new ones where it makes sense
- Don't add unnecessary abstractions — keep it simple
- All file paths are relative to the worktree directory, NOT the main project root
