# Implement Feature

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Execute the implementation plan from `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md`. Write code, write tests, and verify everything builds and passes.

## Tasks

- [ ] **Read the plan**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md`. If the file doesn't exist, mark this task complete without proceeding — there's nothing to implement.

- [ ] **Implement the feature**: Following the plan, create and modify files as specified. Adhere to these rules:
  - Read CLAUDE.md for project conventions
  - Follow existing code patterns and style
  - Do NOT modify CLAUDE.md, any files in `.github/workflows/`, or `scripts/autodev/`
  - Keep files under 500 lines
  - Use `internal/ui` helpers for all output
  - Use the store pattern for data persistence

- [ ] **Write tests**: Create `_test.go` files alongside the code. Cover:
  - Happy path for each new function
  - Error cases and edge cases
  - Any acceptance criteria that can be verified programmatically

- [ ] **Run tests**: Execute `make test` and verify all tests pass (including existing tests). If tests fail, fix the issues before proceeding.

- [ ] **Run build**: Execute `make build` and verify the binary builds successfully. If the build fails, fix the issues.

- [ ] **Verify protected files**: Run `git diff --name-only` and confirm no changes to CLAUDE.md, `.github/workflows/`, or `scripts/autodev/`. If any protected files were modified, revert them with `git checkout -- <file>`.

## Guidelines

- Implement exactly what the plan specifies — don't add unplanned features
- If you discover the plan has a flaw, make a reasonable adjustment and note it
- Run `make test` frequently during implementation, not just at the end
- Prefer editing existing files over creating new ones where it makes sense
- Don't add unnecessary abstractions — keep it simple
