#!/usr/bin/env bash
set -euo pipefail

# scripts/autodev/open-pr.sh — Create a PR for an autodev implementation
#
# Usage:
#   scripts/autodev/open-pr.sh ISSUE_NUMBER BRANCH_NAME
#
# Reads issue details, creates PR with structured body, enables auto-merge.

source "$(dirname "$0")/config.sh"

ISSUE_NUMBER="${1:?Usage: open-pr.sh ISSUE_NUMBER BRANCH_NAME}"
BRANCH_NAME="${2:?Usage: open-pr.sh ISSUE_NUMBER BRANCH_NAME}"

# ── Read issue ─────────────────────────────────────────────────────

ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --json title,body)
ISSUE_TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')

# ── Create PR ──────────────────────────────────────────────────────

PR_URL=$(gh pr create \
    --repo "$AUTODEV_REPO" \
    --head "$BRANCH_NAME" \
    --base "$AUTODEV_BASE_BRANCH" \
    --title "$ISSUE_TITLE" \
    --body "$(cat <<EOF
## Summary

Autonomous implementation of #$ISSUE_NUMBER.

Closes #$ISSUE_NUMBER

## Changes

See commits on this branch for implementation details.

## Test Plan

- [ ] CI passes (\`make test\`, \`make build\`)
- [ ] Code review feedback addressed

<!-- autodev-state: {"iteration": 0} -->
EOF
)" \
    --label "$AUTODEV_LABEL_AUTODEV")

PR_NUMBER=$(gh pr list --repo "$AUTODEV_REPO" --head "$BRANCH_NAME" --json number --jq '.[0].number')

autodev_info "Created PR: $PR_URL"

# ── Trigger CI ─────────────────────────────────────────────────────
# Pushes by GITHUB_TOKEN don't trigger downstream workflows (GitHub
# security policy). Close/reopen the PR to fire the pull_request event
# which triggers CI and code review.

gh pr close "$PR_NUMBER" --repo "$AUTODEV_REPO"
gh pr reopen "$PR_NUMBER" --repo "$AUTODEV_REPO"

autodev_info "Closed/reopened PR #$PR_NUMBER to trigger CI"

# ── Enable auto-merge ──────────────────────────────────────────────

gh pr merge "$PR_NUMBER" --repo "$AUTODEV_REPO" --squash --auto

autodev_info "Auto-merge enabled for PR #$PR_NUMBER"

echo "$PR_URL"
