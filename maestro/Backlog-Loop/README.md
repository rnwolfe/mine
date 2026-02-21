# Backlog Loop Playbook

> **Experimental**: Maestro is not a production implementation path. The two production
> paths are `/autodev` (interactive CLI) and GitHub Actions (always-on pipeline).

A looping Auto Run playbook that picks issues from the GitHub backlog, implements them in isolated worktrees, runs through automated and self-review cycles, updates documentation, and opens review-ready PRs — then repeats until the backlog is empty or the concurrency limit is reached.

## Requirements

- **Agent:** Claude Code
- **Project:** `mine` CLI (Go)
- **Tools:** `gh` CLI authenticated with repo access, `git` with worktree support

## Overview

This playbook automates the full issue-to-PR lifecycle, including code review and documentation. It uses git worktrees for isolation, enabling multiple instances to run in parallel without conflicts.

Each loop runs 9 steps then re-checks for more work:
1. **Picks** the next `backlog/ready` issue (excludes `agent/implementing` and `via/maestro` labeled issues)
2. **Labels** the issue with `via/maestro` + `agent/implementing` to claim it
3. **Plans** the implementation approach in a fresh worktree
4. **Implements** the feature with tests
5. **Opens a PR** with a detailed description
6. **Waits for Copilot review** and addresses any feedback
7. **Self-reviews** with a fresh-context subagent, iterating until clean
8. **Updates documentation** (Starlight site, internal docs) and creates follow-up issues
9. **Finalizes** by labeling PR/issue `human/review-merge` and cleaning up the worktree

After step 9, `9_PROGRESS.md` checks if more issues are available and resets steps 1–8 for the next loop iteration.

## Document Chain

| Document | Purpose | Reset on Completion? |
|----------|---------|---------------------|
| `1_PICK_ISSUE.md` | Select next backlog issue, verify trust, label, create worktree | No |
| `2_PLAN.md` | Read issue, explore codebase in worktree, design approach | No |
| `3_IMPLEMENT.md` | Implement the feature, write tests, verify build (in worktree) | No |
| `4_OPEN_PR.md` | Commit, push, create PR with detailed description | No |
| `5_COPILOT_REVIEW.md` | Wait for Copilot review, address feedback | No |
| `6_SELF_REVIEW.md` | Fresh-context code review loop (up to 3 iterations) | No |
| `7_DOCS_FOLLOWUP.md` | Update docs, create follow-up issues | No |
| `8_FINALIZE.md` | Label `human/review-merge`, remove worktree | No |
| `9_PROGRESS.md` | Check for more issues, reset 1-8 if available, exit if done | **Yes** |

## Recommended Setup

```
Loop Mode: ON
Documents:
  1_PICK_ISSUE.md        [Reset: OFF]  <- Gets reset by 9_PROGRESS
  2_PLAN.md              [Reset: OFF]  <- Gets reset by 9_PROGRESS
  3_IMPLEMENT.md         [Reset: OFF]  <- Gets reset by 9_PROGRESS
  4_OPEN_PR.md           [Reset: OFF]  <- Gets reset by 9_PROGRESS
  5_COPILOT_REVIEW.md    [Reset: OFF]  <- Gets reset by 9_PROGRESS
  6_SELF_REVIEW.md       [Reset: OFF]  <- Gets reset by 9_PROGRESS
  7_DOCS_FOLLOWUP.md     [Reset: OFF]  <- Gets reset by 9_PROGRESS
  8_FINALIZE.md          [Reset: OFF]  <- Gets reset by 9_PROGRESS
  9_PROGRESS.md          [Reset: ON]   <- Controls loop: resets 1-8 if work remains
```

## Parallel Execution

This playbook supports multiple instances running simultaneously:

- **Worktree isolation**: Each issue gets its own git worktree at `{{AGENT_PATH}}-worktrees/issue-ISSUE_NUMBER`, so parallel instances don't conflict on disk
- **Label-based claiming**: Issues are labeled `via/maestro` + `agent/implementing` immediately after selection, preventing other instances from picking the same issue
- **Concurrency limit**: Maximum 3 open `via/maestro` PRs at once (configurable in `1_PICK_ISSUE.md` and `9_PROGRESS.md`)
- **Independent state**: Each loop iteration writes to `LOOP_{{LOOP_NUMBER}}_*` files, so parallel loops don't overwrite each other's state

To run in parallel, start multiple Maestro Auto Run sessions with this playbook. Each will independently pick different issues and work in isolated worktrees.

## Generated Files

Each loop creates:
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` — Selected issue details + worktree path + PR info
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md` — Implementation plan
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_COPILOT.md` — Copilot review status/feedback
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_REVIEW.md` — Self-review summary
- `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md` — Cumulative log of all issues processed

## Labels

| Label | Applied to | When | Purpose |
|-------|-----------|------|---------|
| `backlog/ready` | Issue | Before playbook runs | Signals issue is ready for autonomous work |
| `via/maestro` | Issue | On pick (step 1) | Claims issue for this playbook |
| `agent/implementing` | Issue | On pick (step 1) | Prevents re-selection by parallel instances |
| `via/maestro` | PR | On creation (step 4) | Identifies maestro-created PRs |
| `human/review-merge` | PR + Issue | On finalize (step 8) | Signals human review can begin |

## Safety

- Only processes issues labeled by trusted users (checked via timeline API)
- Protected files (CLAUDE.md, workflows, autodev scripts) are never modified
- Max 3 open `via/maestro` PRs at a time (concurrency guard)
- Each implementation runs `make test` and `make build` before committing
- Worktrees are cleaned up after each loop to avoid disk accumulation
- Self-review loop has a max of 3 iterations to prevent infinite cycles

## Template Variables

- `{{AGENT_NAME}}` — Maestro agent name
- `{{AGENT_PATH}}` — Project root path
- `{{AUTORUN_FOLDER}}` — Path to this playbook folder
- `{{LOOP_NUMBER}}` — Current loop iteration
- `{{DATE}}` — Today's date (YYYY-MM-DD)
