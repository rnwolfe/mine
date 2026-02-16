# Backlog Loop Playbook

A looping Auto Run playbook that picks issues from the GitHub backlog, implements them, opens PRs, and repeats until the backlog is empty or the concurrency limit is reached.

## Requirements

- **Agent:** Claude Code
- **Project:** `mine` CLI (Go)
- **Tools:** `gh` CLI authenticated with repo access

## Overview

This playbook automates the same workflow as the `autodev-dispatch` + `autodev-implement` GitHub Actions pipeline, but driven locally via Maestro for faster iteration and developer oversight.

Each loop:
1. **Picks** the next `agent-ready` issue from the backlog
2. **Plans** the implementation approach
3. **Implements** the feature with tests
4. **Opens a PR** with a detailed description
5. **Checks progress** and loops if more issues are available

## Document Chain

| Document | Purpose | Reset on Completion? |
|----------|---------|---------------------|
| `1_PICK_ISSUE.md` | Select next backlog issue, verify trust, create branch | No |
| `2_PLAN.md` | Read issue, explore codebase, design implementation approach | No |
| `3_IMPLEMENT.md` | Implement the feature, write tests, verify build | No |
| `4_OPEN_PR.md` | Commit, push, create PR with detailed description | No |
| `5_PROGRESS.md` | Check for more issues, reset 1-4 if available, exit if done | **Yes** |

## Recommended Setup

```
Loop Mode: ON
Documents:
  1_PICK_ISSUE.md   [Reset: OFF]  ← Gets reset by 5_PROGRESS
  2_PLAN.md         [Reset: OFF]  ← Gets reset by 5_PROGRESS
  3_IMPLEMENT.md    [Reset: OFF]  ← Gets reset by 5_PROGRESS
  4_OPEN_PR.md      [Reset: OFF]  ← Gets reset by 5_PROGRESS
  5_PROGRESS.md     [Reset: ON]   ← Controls loop: resets 1-4 if work remains
```

## Generated Files

Each loop creates:
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` — Selected issue details
- `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md` — Implementation plan
- `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md` — Cumulative log of all issues processed

## Safety

- Only processes issues labeled by trusted users (checked via timeline API)
- Protected files (CLAUDE.md, workflows, autodev scripts) are never modified
- Max 1 open autodev PR at a time (concurrency guard)
- Each implementation runs `make test` and `make build` before committing

## Template Variables

- `{{AGENT_NAME}}` — Maestro agent name
- `{{AGENT_PATH}}` — Project root path
- `{{AUTORUN_FOLDER}}` — Path to this playbook folder
- `{{LOOP_NUMBER}}` — Current loop iteration
- `{{DATE}}` — Today's date (YYYY-MM-DD)
