# Self-Review Loop

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Perform a thorough code review of the implementation using a fresh context (subagent), post the review as a PR comment, address any issues found, and repeat until the review signals the code is ready for human review. Maximum 3 review iterations.

## Tasks

- [ ] **Read PR and worktree details**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and extract the PR number from `## PR` and the worktree path from `## Worktree`. If no PR section exists, mark complete without proceeding.

- [ ] **Self-review iteration loop**: Perform the following cycle up to 3 times. Track the iteration count. For each iteration:

  ### 1. Gather context for review

  Collect the full diff of all changes against main:
  ```
  git -C WORKTREE_PATH diff origin/main...HEAD
  ```

  Also collect the list of changed files:
  ```
  git -C WORKTREE_PATH diff origin/main...HEAD --name-only
  ```

  Read the issue body and implementation plan for context on what was intended.

  ### 2. Perform fresh-context review via subagent

  Use the **Task tool** to spawn a fresh subagent (subagent_type: `general-purpose`) with a prompt like:

  > You are a senior Go developer reviewing a pull request for the `mine` CLI tool.
  > Review the following diff for:
  > - **Correctness**: Logic errors, off-by-one, nil pointer risks, unchecked errors
  > - **Go idioms**: Proper error handling, naming conventions, interface usage
  > - **Project conventions**: Domain separation (thin cmd/, logic in internal/), UI helpers for output, store pattern for data, files under 500 lines
  > - **Tests**: Coverage gaps, missing edge cases, test quality
  > - **Security**: Input validation, path traversal, injection risks
  > - **Simplicity**: Over-engineering, unnecessary abstractions, dead code
  >
  > For each issue found, specify: severity (critical/warning/nit), file path, line reference, and what to change.
  >
  > If the code is clean and ready for human review, explicitly state: "REVIEW_CLEAN: This code is ready for human review."
  >
  > Context — Issue: <issue title and body summary>
  > Plan: <plan summary>
  >
  > Diff:
  > <full diff>

  The subagent should return a structured review.

  ### 3. Post review as PR comment

  Post the subagent's review as a PR comment:
  ```
  gh pr comment PR_NUMBER --repo rnwolfe/mine --body "<formatted review>"
  ```

  Prefix the comment with `## Self-Review (Iteration N)` so it's clearly identifiable.

  ### 4. Check for clean signal

  If the review contains "REVIEW_CLEAN" or has no critical/warning findings, **exit the loop** — the code is ready. Proceed to the next document.

  ### 5. Address review findings

  For each critical and warning finding:
  1. Read the relevant file in the worktree
  2. Make the fix
  3. Verify with `make test && make build` from the worktree

  Nits are optional — fix them if quick, skip if not.

  ### 6. Commit and push fixes

  ```
  git -C WORKTREE_PATH add -A
  git -C WORKTREE_PATH commit -m "fix: address self-review findings (iteration N)"
  git -C WORKTREE_PATH push
  ```

  ### 7. Continue or exit

  If this was iteration 3 (max reached), post a final PR comment noting any remaining issues and exit the loop. Otherwise, continue to the next iteration.

- [ ] **Write review summary**: Save the final review status to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_REVIEW.md`:
  ```markdown
  # Self-Review Summary

  ## Iterations: N
  ## Final Status: CLEAN | REMAINING_ISSUES

  ## Findings Addressed
  - <list of issues fixed across all iterations>

  ## Remaining Issues (if any)
  - <list of issues not fully resolved>
  ```

## Guidelines

- The subagent review MUST use a fresh context — do not reuse your current analysis. The whole point is an independent second opinion.
- Post reviews as PR comments so they're visible to human reviewers later
- Critical findings must be fixed. Warnings should be fixed. Nits are optional.
- If the first review is clean, only one iteration is needed — don't force unnecessary cycles.
- All code changes happen in the worktree directory
- Run `make test` after every batch of fixes, not just at the end
