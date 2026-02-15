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

PR_JSON=$(gh pr create \
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
    --label "$AUTODEV_LABEL_AUTODEV" \
    --json number,url)

PR_NUMBER=$(echo "$PR_JSON" | jq -r '.number')
PR_URL=$(echo "$PR_JSON" | jq -r '.url')

autodev_info "Created PR: $PR_URL"

# ── Enable auto-merge ──────────────────────────────────────────────

gh pr merge "$PR_NUMBER" --repo "$AUTODEV_REPO" --squash --auto

autodev_info "Auto-merge enabled for PR #$PR_NUMBER"

echo "$PR_URL"
