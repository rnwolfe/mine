# Open Pull Request

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Commit all changes, push the branch, and create a detailed PR. The PR description should be comprehensive enough for a human reviewer to understand exactly what was built and why.

## Tasks

- [ ] **Check for changes**: Run `git diff --stat` and `git diff --cached --stat`. If there are no changes (both empty), write "SKIPPED: no changes to commit" to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md` and mark complete without proceeding.

- [ ] **Read issue details**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` to get the issue number and title. Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md` for the implementation plan.

- [ ] **Commit changes**: Stage and commit all changes:
  ```
  git add -A
  git commit -m "feat: implement #ISSUE_NUMBER"
  ```
  Do NOT commit files that contain secrets or credentials.

- [ ] **Push branch**: Push the branch to origin:
  ```
  git push -u origin BRANCH_NAME
  ```

- [ ] **Create PR with detailed description**: Create the PR using `gh pr create` with a comprehensive body. The PR must include:

  **Title:** The issue title (from the issue file)

  **Body structure:**
  ```markdown
  ## Summary

  <2-4 sentence overview of what was implemented and why.
   Reference the design approach from the plan.>

  Closes #ISSUE_NUMBER

  ## Changes

  <Bulleted list of all changes, organized by area:>
  - **New files**: each new file and its purpose
  - **Modified files**: each modified file and what changed
  - **Architecture**: design decisions and patterns used

  ## CLI Surface

  <New commands, subcommands, and flags added.
   Include usage examples.>
  - `mine <command>` — description
  - Flags: `--flag` — description

  ## Test Coverage

  <What tests were added:>
  - Unit tests for ...
  - Edge cases covered: ...

  ## Acceptance Criteria

  <Verified against issue #N:>
  - [x] Criterion — how it was met
  - [ ] Criterion — why it wasn't met (if any)
  ```

  Add the `autodev` label: `--label autodev`

  Append the autodev state tracker at the end of the body:
  `<!-- autodev-state: {"phase": "copilot", "copilot_iterations": 0} -->`

- [ ] **Log the PR**: Append to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md`:
  ```markdown
  ## Loop {{LOOP_NUMBER}} — Issue #N: Title
  - **Branch:** branch-name
  - **PR:** PR_URL
  - **Status:** PR opened
  - **Files changed:** N
  ```

## Guidelines

- The PR description is critical — it's the first thing reviewers see. Be thorough.
- Use conventional commit format for the commit message (`feat:`, `fix:`, etc.)
- Don't force push or rewrite history
- Verify the PR was created successfully before marking complete
