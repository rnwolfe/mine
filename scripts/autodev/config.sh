#!/usr/bin/env bash
# scripts/autodev/config.sh — Shared constants for autodev workflows
#
# Source this file from other autodev scripts:
#   source "$(dirname "$0")/config.sh"

# Repository
AUTODEV_REPO="rnwolfe/mine"
AUTODEV_BASE_BRANCH="main"

# Labels
AUTODEV_LABEL_READY="agent-ready"
AUTODEV_LABEL_IN_PROGRESS="in-progress"
AUTODEV_LABEL_AUTODEV="autodev"
AUTODEV_LABEL_NEEDS_HUMAN="needs-human"

# Limits
AUTODEV_MAX_ITERATIONS=3
AUTODEV_MAX_OPEN_PRS=1

# Provider (model-agnostic switch)
AUTODEV_PROVIDER="${AUTODEV_PROVIDER:-claude}"

# ── Logging helpers ─────────────────────────────────────────────────

autodev_info()  { echo "::notice::autodev: $*"; }
autodev_warn()  { echo "::warning::autodev: $*"; }
autodev_error() { echo "::error::autodev: $*"; }

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
