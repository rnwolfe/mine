# Finalize — Label and Clean Up

## Context

- **Playbook:** Backlog Loop
- **Agent:** Mine CLI
- **Project:** /home/rnwolfe/dev/mine
- **Auto Run Folder:** /home/rnwolfe/dev/mine/maestro
- **Loop:** 00001

## Objective

Mark the PR and issue as ready for human review by applying the `maestro/review-ready` label. Clean up the git worktree. This signals to the maintainer that autonomous work is complete and the PR is ready for final review and merge.

## Tasks

- [x] **Read PR and issue details**: Read `/home/rnwolfe/dev/mine/maestro/LOOP_00001_ISSUE.md` and extract the issue number, PR number, and worktree path. If no PR section exists (issue was blocked), skip to the cleanup step.
  - Issue #19, PR #107, worktree `/home/rnwolfe/dev/mine-worktrees/issue-19`

- [x] **Label PR as review-ready**: Add the `maestro/review-ready` label to the PR:
  ```
  gh pr edit PR_NUMBER --repo rnwolfe/mine --add-label "maestro/review-ready"
  ```
  - Applied `maestro/review-ready` to PR #107

- [x] **Label issue as review-ready**: Add the `maestro/review-ready` label to the issue:
  ```
  gh issue edit ISSUE_NUMBER --repo rnwolfe/mine --add-label "maestro/review-ready"
  ```
  - Applied `maestro/review-ready` to issue #19

- [x] **Remove worktree**: Clean up the git worktree used for this implementation:
  ```
  git -C /home/rnwolfe/dev/mine worktree remove /home/rnwolfe/dev/mine-worktrees/issue-ISSUE_NUMBER --force
  ```
  If the worktree directory doesn't exist (e.g., issue was blocked before worktree creation), skip this step. Also clean up the worktrees parent directory if empty:
  ```
  rmdir /home/rnwolfe/dev/mine-worktrees 2>/dev/null || true
  ```
  - Worktree removed and parent directory cleaned up

- [x] **Update backlog log**: Append to `/home/rnwolfe/dev/mine/maestro/BACKLOG_LOG_2026-02-17.md`:
  ```markdown
  ### Loop 00001 — Finalized
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
