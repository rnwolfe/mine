#!/usr/bin/env bash
# scripts/autodev/config.sh — Shared constants for autodev workflows
#
# Source this file from other autodev scripts:
#   source "$(dirname "$0")/config.sh"

# Repository
AUTODEV_REPO="rnwolfe/mine"
AUTODEV_BASE_BRANCH="main"

# Pipeline stage labels (mutually exclusive per issue/PR)
AUTODEV_LABEL_READY="backlog/ready"
AUTODEV_LABEL_IMPLEMENTING="agent/implementing"
AUTODEV_LABEL_REVIEW_COPILOT="agent/review-copilot"
AUTODEV_LABEL_REVIEW_CLAUDE="agent/review-claude"
AUTODEV_LABEL_REVIEW_MERGE="human/review-merge"
AUTODEV_LABEL_BLOCKED="human/blocked"

# Origin labels (persistent, one per PR)
AUTODEV_LABEL_VIA_ACTIONS="via/actions"
AUTODEV_LABEL_VIA_AUTODEV="via/autodev"

# Priority labels (dispatch ordering)
AUTODEV_LABEL_PRIORITY_CRITICAL="priority/critical"
AUTODEV_LABEL_PRIORITY_HIGH="priority/high"

# Report labels
AUTODEV_LABEL_PIPELINE_AUDIT="report/pipeline-audit"

# Limits
AUTODEV_MAX_ITERATIONS=3

# Trusted users who can trigger autodev via backlog/ready label
# Only these users (repo owner/collaborators) can queue work for autonomous execution.
# Bot actors are included so autodev-created follow-up issues can be auto-queued.
AUTODEV_TRUSTED_USERS=("rnwolfe" "github-actions[bot]" "mine-autodev[bot]" "claude[bot]")

# Human reviewer to remove from autodev PRs (auto-added by CODEOWNERS).
# Autodev PRs are reviewed by Copilot and Claude only; humans are looped in
# exclusively via human/blocked or human/review-merge guardrails.
AUTODEV_HUMAN_REVIEWER="rnwolfe"

# Provider (model-agnostic switch)
AUTODEV_PROVIDER="${AUTODEV_PROVIDER:-claude}"

# ── Logging helpers ─────────────────────────────────────────────────

autodev_info()  { echo "::notice::autodev: $*" >&2; }
autodev_warn()  { echo "::warning::autodev: $*" >&2; }
autodev_error() { echo "::error::autodev: $*" >&2; }

autodev_fatal() {
    autodev_error "$@"
    exit 1
}

# ── Utilities ───────────────────────────────────────────────────────

# Convert a string to a branch-safe slug
# Usage: autodev_slugify "Add mine about command"  →  add-mine-about-command
autodev_slugify() {
    echo "$1" \
        | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9]+/-/g' \
        | sed -E 's/^-+|-+$//g' \
        | cut -c1-50
}
