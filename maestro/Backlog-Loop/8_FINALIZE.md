# Finalize — Label and Clean Up

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Mark the PR and issue as ready for human review by applying the `maestro/review-ready` label. Clean up the git worktree. This signals to the maintainer that autonomous work is complete and the PR is ready for final review and merge.

## Tasks

- [ ] **Read PR and issue details**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and extract the issue number, PR number, and worktree path. If no PR section exists (issue was blocked), skip to the cleanup step.

- [ ] **Label PR as review-ready**: Add the `maestro/review-ready` label to the PR:
  ```
  gh pr edit PR_NUMBER --repo rnwolfe/mine --add-label "maestro/review-ready"
  ```

- [ ] **Label issue as review-ready**: Add the `maestro/review-ready` label to the issue:
  ```
  gh issue edit ISSUE_NUMBER --repo rnwolfe/mine --add-label "maestro/review-ready"
  ```

- [ ] **Remove worktree**: Clean up the git worktree used for this implementation:
  ```
  git -C {{AGENT_PATH}} worktree remove {{AGENT_PATH}}-worktrees/issue-ISSUE_NUMBER --force
  ```
  If the worktree directory doesn't exist (e.g., issue was blocked before worktree creation), skip this step. Also clean up the worktrees parent directory if empty:
  ```
  rmdir {{AGENT_PATH}}-worktrees 2>/dev/null || true
  ```

- [ ] **Update backlog log**: Append to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md`:
  ```markdown
  ### Loop {{LOOP_NUMBER}} — Finalized
  - **Issue:** #ISSUE_NUMBER
  - **PR:** #PR_NUMBER
  - **Status:** maestro/review-ready — awaiting human review
  - **Worktree:** cleaned up
  ```

## Guidelines

- The `maestro/review-ready` label signals to the maintainer that:
  1. Implementation is complete
  2. Copilot review feedback has been addressed
  3. Self-review loop passed
  4. Documentation is up to date
  5. Follow-up issues have been created
- Do NOT remove the `in-progress` or `maestro` labels — the maintainer will remove those when merging
- Always clean up worktrees to avoid disk space accumulation across loops
- If the worktree remove fails (dirty state), force-remove it — the branch and commits are already pushed
