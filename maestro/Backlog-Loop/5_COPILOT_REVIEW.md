# Wait for and Address Copilot Review

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Wait for GitHub Copilot's automated code review on the PR, then address any feedback it provides. If Copilot doesn't run or has no comments, proceed to the next step. This ensures the PR is clean before the self-review phase.

## Tasks

- [ ] **Read PR details**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and extract the PR number from the `## PR` section, and the worktree path from `## Worktree`. If no PR section exists, mark complete without proceeding.

- [ ] **Wait for Copilot review**: Poll for Copilot's review using:
  ```
  gh api repos/rnwolfe/mine/pulls/PR_NUMBER/reviews \
    --jq '[.[] | select(.user.login == "copilot-pull-request-reviewer[bot]" or .user.login == "github-copilot[bot]")] | length'
  ```
  Poll every 30 seconds for up to 10 minutes (20 attempts). If Copilot doesn't post a review within the timeout, write "COPILOT_SKIPPED: no review posted within timeout" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_COPILOT.md` and proceed to the next task.

- [ ] **Extract Copilot feedback**: Once a Copilot review is detected, extract its comments:

  **Top-level review body:**
  ```
  gh api repos/rnwolfe/mine/pulls/PR_NUMBER/reviews \
    --jq '[.[] | select(.user.login == "copilot-pull-request-reviewer[bot]" or .user.login == "github-copilot[bot]")] | last | {body: .body, state: .state, id: .id}'
  ```

  **Inline comments from Copilot:**
  ```
  gh api repos/rnwolfe/mine/pulls/PR_NUMBER/comments \
    --jq '[.[] | select(.user.login == "copilot-pull-request-reviewer[bot]" or .user.login == "github-copilot[bot]")] | .[] | {path: .path, line: .line, body: .body, id: .id}'
  ```

  If there are no inline comments and the review body is empty or just a generic approval, write "COPILOT_CLEAN: no actionable feedback" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_COPILOT.md` and proceed.

  Otherwise, save the feedback to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_COPILOT.md` in structured format:
  ```markdown
  # Copilot Review Feedback

  ## Review Body
  <body text>

  ## Inline Comments
  ### path/to/file.go:42 [comment_id: ID]
  <comment body>
  ```

- [ ] **Address Copilot feedback**: For each actionable Copilot comment:
  1. Read the relevant code in the **worktree directory**
  2. Make the fix in the worktree
  3. Reply to the inline comment:
     ```
     gh api "repos/rnwolfe/mine/pulls/PR_NUMBER/comments/COMMENT_ID/replies" -f body="Fixed: <explanation>"
     ```
  4. If you disagree with a suggestion, reply explaining why

- [ ] **Run tests after fixes**: If any code changes were made, run `make test && make build` from the worktree directory. Fix any failures.

- [ ] **Commit and push Copilot fixes**: If changes were made:
  ```
  git -C WORKTREE_PATH add -A
  git -C WORKTREE_PATH commit -m "fix: address copilot review feedback"
  git -C WORKTREE_PATH push
  ```
  If no changes were needed, skip this step.

## Guidelines

- Copilot reviews use state `COMMENTED`, not `changes_requested` — check the reviewer identity, not the state
- Don't block forever waiting for Copilot — the 10 minute timeout ensures progress
- If Copilot's suggestions conflict with project conventions (CLAUDE.md), prefer the project conventions and explain why in your reply
- All code changes happen in the worktree, not the main checkout
