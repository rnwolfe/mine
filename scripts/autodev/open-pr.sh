#!/usr/bin/env bash
set -euo pipefail

# scripts/autodev/open-pr.sh — Create a PR for an autodev implementation
#
# Usage:
#   scripts/autodev/open-pr.sh ISSUE_NUMBER BRANCH_NAME [ORIGIN_LABEL]
#
# Uses agent-generated PR description from /tmp/pr-description.md if available,
# otherwise falls back to a basic template.
# Human merges manually after reviewing.

source "$(dirname "$0")/config.sh"

ISSUE_NUMBER="${1:?Usage: open-pr.sh ISSUE_NUMBER BRANCH_NAME}"
BRANCH_NAME="${2:?Usage: open-pr.sh ISSUE_NUMBER BRANCH_NAME}"
ORIGIN_LABEL="${3:-$AUTODEV_LABEL_VIA_ACTIONS}"

# ── Read issue ─────────────────────────────────────────────────────

ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --json title,body)
ISSUE_TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')

# ── Read agent-generated PR title ─────────────────────────────────

PR_TITLE_FILE="/tmp/pr-title.txt"
if [ -f "$PR_TITLE_FILE" ] && [ -s "$PR_TITLE_FILE" ]; then
    autodev_info "Using agent-generated PR title"
    PR_TITLE=$(head -1 "$PR_TITLE_FILE")
else
    autodev_warn "No agent-generated PR title found, falling back to issue title"
    PR_TITLE="$ISSUE_TITLE"
fi

# ── Build PR body ─────────────────────────────────────────────────

PR_DESC_FILE="/tmp/pr-description.md"

if [ -f "$PR_DESC_FILE" ] && [ -s "$PR_DESC_FILE" ]; then
    autodev_info "Using agent-generated PR description"
    PR_BODY=$(cat "$PR_DESC_FILE")
else
    autodev_warn "No agent-generated PR description found, using fallback template"
    PR_BODY=$(cat <<EOF
## Summary

Autonomous implementation of #$ISSUE_NUMBER.

Closes #$ISSUE_NUMBER

## Changes

See commits on this branch for implementation details.

## Test Plan

- [ ] CI passes (\`make test\`, \`make build\`)
- [ ] Code review feedback addressed
EOF
)
fi

# Append autodev state tracker
PR_BODY+=$'\n\n<!-- autodev-state: {"phase": "copilot", "copilot_iterations": 0} -->'

# ── Create PR ──────────────────────────────────────────────────────

PR_URL=$(gh pr create \
    --repo "$AUTODEV_REPO" \
    --head "$BRANCH_NAME" \
    --base "$AUTODEV_BASE_BRANCH" \
    --title "$PR_TITLE" \
    --body "$PR_BODY" \
    --label "$ORIGIN_LABEL" --label "$AUTODEV_LABEL_REVIEW_COPILOT")

autodev_info "Created PR: $PR_URL"

# Explicitly request Copilot review — GitHub skips auto-review for bot-authored PRs,
# so we must request it manually to keep the pipeline flowing.
PR_NUMBER=$(echo "$PR_URL" | grep -oP '/pull/\K\d+')
if gh pr edit "$PR_NUMBER" \
    --repo "$AUTODEV_REPO" \
    --add-reviewer "copilot-pull-request-reviewer[bot]" 2>/dev/null; then
    autodev_info "Requested Copilot review on PR #$PR_NUMBER"
else
    autodev_warn "Could not request Copilot review on PR #$PR_NUMBER — pipeline will fall back to Claude phase via scheduled poll"
fi

echo "$PR_URL"
