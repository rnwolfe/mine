#!/usr/bin/env bash
set -euo pipefail

# scripts/autodev/pick-issue.sh — Select the next issue for autodev
#
# Usage:
#   scripts/autodev/pick-issue.sh [ISSUE_NUMBER]
#
# If ISSUE_NUMBER is provided, uses that issue (manual override).
# Otherwise, picks the oldest agent-ready issue not already in-progress.
# Exits 0 with empty output if nothing to do.

source "$(dirname "$0")/config.sh"

ISSUE_NUMBER="${1:-}"

# ── Check concurrency limit ────────────────────────────────────────

OPEN_PRS=$(gh pr list \
    --repo "$AUTODEV_REPO" \
    --label "$AUTODEV_LABEL_AUTODEV" \
    --state open \
    --json number \
    --jq 'length')

if [ "$OPEN_PRS" -ge "$AUTODEV_MAX_OPEN_PRS" ]; then
    autodev_info "Concurrency limit reached ($OPEN_PRS/$AUTODEV_MAX_OPEN_PRS autodev PRs open). Skipping."
    exit 0
fi

# ── Pick an issue ──────────────────────────────────────────────────

if [ -n "$ISSUE_NUMBER" ]; then
    # Manual override — validate the issue exists and has agent-ready label
    LABELS=$(gh issue view "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --json labels --jq '[.labels[].name]' 2>/dev/null) \
        || autodev_fatal "Issue #$ISSUE_NUMBER not found"

    if ! echo "$LABELS" | jq -e --arg label "$AUTODEV_LABEL_READY" 'index($label)' >/dev/null 2>&1; then
        autodev_fatal "Issue #$ISSUE_NUMBER does not have '$AUTODEV_LABEL_READY' label"
    fi

    # Label in-progress atomically to prevent race conditions
    gh issue edit "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --add-label "$AUTODEV_LABEL_IN_PROGRESS" >/dev/null

    echo "$ISSUE_NUMBER"
    exit 0
fi

# Auto-pick: oldest agent-ready issue not in-progress
ISSUE_NUMBER=$(gh issue list \
    --repo "$AUTODEV_REPO" \
    --label "$AUTODEV_LABEL_READY" \
    --state open \
    --json number,labels \
    --jq "[.[] | select(.labels | map(.name) | index(\"$AUTODEV_LABEL_IN_PROGRESS\") | not)] | sort_by(.number) | first | .number // empty")

if [ -z "$ISSUE_NUMBER" ]; then
    autodev_info "No agent-ready issues found. Nothing to do."
    exit 0
fi

# Label in-progress atomically to prevent race conditions
gh issue edit "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --add-label "$AUTODEV_LABEL_IN_PROGRESS" >/dev/null

echo "$ISSUE_NUMBER"
