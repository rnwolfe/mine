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

# ── Verify agent-ready label was applied by a trusted user ────────

verify_trusted_labeler() {
    local issue="$1"

    # Get who applied the agent-ready label via timeline API
    LABELER=$(gh api "repos/$AUTODEV_REPO/issues/$issue/timeline" --paginate \
        --jq "[.[] | select(.event == \"labeled\" and .label.name == \"$AUTODEV_LABEL_READY\")] | last | .actor.login // empty")

    if [ -z "$LABELER" ]; then
        autodev_warn "Could not determine who applied '$AUTODEV_LABEL_READY' label to issue #$issue"
        return 1
    fi

    for trusted in "${AUTODEV_TRUSTED_USERS[@]}"; do
        if [ "$LABELER" = "$trusted" ]; then
            autodev_info "Label applied by trusted user: $LABELER"
            return 0
        fi
    done

    autodev_warn "Issue #$issue: '$AUTODEV_LABEL_READY' label applied by untrusted user '$LABELER'. Skipping."
    return 1
}

# ── Pick an issue ──────────────────────────────────────────────────

if [ -n "$ISSUE_NUMBER" ]; then
    # Manual override — validate the issue exists and has agent-ready label
    LABELS=$(gh issue view "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --json labels --jq '[.labels[].name]' 2>/dev/null) \
        || autodev_fatal "Issue #$ISSUE_NUMBER not found"

    if ! echo "$LABELS" | jq -e --arg label "$AUTODEV_LABEL_READY" 'index($label)' >/dev/null 2>&1; then
        autodev_fatal "Issue #$ISSUE_NUMBER does not have '$AUTODEV_LABEL_READY' label"
    fi

    # Verify label was applied by a trusted user
    verify_trusted_labeler "$ISSUE_NUMBER" \
        || autodev_fatal "Issue #$ISSUE_NUMBER: agent-ready label not applied by a trusted user"

    # Label in-progress atomically to prevent race conditions
    gh issue edit "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --add-label "$AUTODEV_LABEL_IN_PROGRESS" >/dev/null

    echo "$ISSUE_NUMBER"
    exit 0
fi

# Auto-pick: oldest agent-ready issue not in-progress
CANDIDATES=$(gh issue list \
    --repo "$AUTODEV_REPO" \
    --label "$AUTODEV_LABEL_READY" \
    --state open \
    --json number,labels \
    --jq "[.[] | select(.labels | map(.name) | index(\"$AUTODEV_LABEL_IN_PROGRESS\") | not)] | sort_by(.number)")

# Find the first candidate with a trusted labeler
ISSUE_NUMBER=""
for row in $(echo "$CANDIDATES" | jq -r '.[].number'); do
    if verify_trusted_labeler "$row"; then
        ISSUE_NUMBER="$row"
        break
    fi
done

if [ -z "$ISSUE_NUMBER" ]; then
    autodev_info "No agent-ready issues from trusted users found. Nothing to do."
    exit 0
fi

# Label in-progress atomically to prevent race conditions
gh issue edit "$ISSUE_NUMBER" --repo "$AUTODEV_REPO" --add-label "$AUTODEV_LABEL_IN_PROGRESS" >/dev/null

echo "$ISSUE_NUMBER"
