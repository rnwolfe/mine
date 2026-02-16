# Review Fix Playbook

A looping Auto Run playbook that addresses PR review feedback, replies to comments, and iterates until reviews are clean.

## Requirements

- **Agent:** Claude Code
- **Project:** `mine` CLI (Go)
- **Tools:** `gh` CLI authenticated with repo access

## Overview

This replaces the `autodev-review-fix` GitHub Actions workflow for local use. It processes review feedback on an autodev PR, fixes issues, replies to comments, and loops until no actionable feedback remains.

Each loop:
1. **Extracts** review feedback from the PR
2. **Fixes** the issues and replies to each comment
3. **Pushes** the changes
4. **Waits** for re-review and checks if more feedback exists

## Setup

Before running, set the PR number in `1_EXTRACT_FEEDBACK.md` by replacing `PR_NUMBER` with the actual number.

## Document Chain

| Document | Purpose | Reset on Completion? |
|----------|---------|---------------------|
| `1_EXTRACT_FEEDBACK.md` | Parse review comments from the PR | No |
| `2_FIX_AND_REPLY.md` | Address each comment, reply on GitHub | No |
| `3_PUSH.md` | Commit, push, check for more feedback | **Yes** |

## Recommended Setup

```
Loop Mode: ON
Max Loops: 3
Documents:
  1_EXTRACT_FEEDBACK.md  [Reset: OFF]  ← Gets reset by 3_PUSH
  2_FIX_AND_REPLY.md     [Reset: OFF]  ← Gets reset by 3_PUSH
  3_PUSH.md              [Reset: ON]   ← Controls loop
```
