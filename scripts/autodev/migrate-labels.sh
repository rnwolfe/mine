#!/usr/bin/env bash
set -euo pipefail

# scripts/autodev/migrate-labels.sh — Migrate GitHub labels to new taxonomy
#
# Usage:
#   scripts/autodev/migrate-labels.sh create    # Create all new labels (idempotent)
#   scripts/autodev/migrate-labels.sh migrate   # Move issues/PRs from old → new labels
#   scripts/autodev/migrate-labels.sh cleanup   # Delete old labels
#
# Run in order: create → migrate → cleanup

source "$(dirname "$0")/config.sh"

# ── Old → New label mapping ──────────────────────────────────────

declare -A LABEL_MAP=(
    ["agent-ready"]="backlog/ready"
    ["needs-refinement"]="backlog/needs-refinement"
    ["in-progress"]="agent/implementing"
    ["claude-review-requested"]="agent/review-claude"
    ["needs-human"]="human/blocked"
    ["autodev"]="via/autodev"
    ["maestro"]="via/maestro"
    ["maestro/review-ready"]="human/review-merge"
    ["autodev-audit"]="report/pipeline-audit"
)

# ── New labels with colors ────────────────────────────────────────

declare -A NEW_LABELS=(
    # backlog/* — light blue (#C5DEF5)
    ["backlog/ready"]="#C5DEF5"
    ["backlog/needs-refinement"]="#C5DEF5"
    ["backlog/triage"]="#C5DEF5"
    ["backlog/needs-spec"]="#C5DEF5"

    # agent/* — salmon (#E99695)
    ["agent/implementing"]="#E99695"
    ["agent/review-copilot"]="#E99695"
    ["agent/review-claude"]="#E99695"

    # human/* — green (#0E8A16)
    ["human/blocked"]="#0E8A16"
    ["human/review-merge"]="#0E8A16"

    # via/* — grey (#EDEDED)
    ["via/autodev"]="#EDEDED"
    ["via/actions"]="#EDEDED"
    ["via/maestro"]="#EDEDED"

    # report/* — purple (#D876E3)
    ["report/pipeline-audit"]="#D876E3"
)

# ── Label descriptions ────────────────────────────────────────────

declare -A LABEL_DESCRIPTIONS=(
    ["backlog/ready"]="Issue is ready for autonomous implementation"
    ["backlog/needs-refinement"]="Issue needs further refinement before implementation"
    ["backlog/triage"]="New issue, needs evaluation"
    ["backlog/needs-spec"]="Passed eval, needs specification"
    ["agent/implementing"]="Agent is actively implementing this issue"
    ["agent/review-copilot"]="Agent addressing Copilot feedback"
    ["agent/review-claude"]="Agent addressing Claude review feedback"
    ["human/blocked"]="Automation hit a limit, needs human intervention"
    ["human/review-merge"]="All automated reviews done, needs human merge"
    ["via/autodev"]="Created by /autodev CLI skill"
    ["via/actions"]="Created by GitHub Actions pipeline"
    ["via/maestro"]="Created by maestro orchestration"
    ["report/pipeline-audit"]="Weekly pipeline health report"
)

# ── Subcommands ───────────────────────────────────────────────────

cmd_create() {
    autodev_info "Creating new labels (idempotent via --force)..."

    for label in "${!NEW_LABELS[@]}"; do
        local color="${NEW_LABELS[$label]}"
        local desc="${LABEL_DESCRIPTIONS[$label]:-}"
        autodev_info "  Creating label: $label ($color)"
        gh label create "$label" \
            --repo "$AUTODEV_REPO" \
            --color "${color#\#}" \
            --description "$desc" \
            --force
    done

    autodev_info "All labels created."
}

cmd_migrate() {
    autodev_info "Migrating issues/PRs from old labels to new labels..."

    for old_label in "${!LABEL_MAP[@]}"; do
        local new_label="${LABEL_MAP[$old_label]}"
        autodev_info "  Migrating: $old_label → $new_label"

        # Find open issues with the old label
        local issues
        issues=$(gh issue list \
            --repo "$AUTODEV_REPO" \
            --label "$old_label" \
            --state all \
            --json number \
            --jq '.[].number' 2>/dev/null) || true

        for issue in $issues; do
            autodev_info "    Issue #$issue: adding '$new_label', removing '$old_label'"
            gh issue edit "$issue" \
                --repo "$AUTODEV_REPO" \
                --add-label "$new_label" \
                --remove-label "$old_label" || true
        done

        # Find PRs with the old label
        local prs
        prs=$(gh pr list \
            --repo "$AUTODEV_REPO" \
            --label "$old_label" \
            --state all \
            --json number \
            --jq '.[].number' 2>/dev/null) || true

        for pr in $prs; do
            autodev_info "    PR #$pr: adding '$new_label', removing '$old_label'"
            gh pr edit "$pr" \
                --repo "$AUTODEV_REPO" \
                --add-label "$new_label" \
                --remove-label "$old_label" || true
        done
    done

    autodev_info "Migration complete."
}

cmd_cleanup() {
    autodev_info "Deleting old labels..."

    for old_label in "${!LABEL_MAP[@]}"; do
        autodev_info "  Deleting: $old_label"
        gh label delete "$old_label" \
            --repo "$AUTODEV_REPO" \
            --yes 2>/dev/null || autodev_warn "  Label '$old_label' not found (already deleted?)"
    done

    autodev_info "Cleanup complete."
}

# ── Main ──────────────────────────────────────────────────────────

case "${1:-}" in
    create)  cmd_create  ;;
    migrate) cmd_migrate ;;
    cleanup) cmd_cleanup ;;
    *)
        echo "Usage: $(basename "$0") {create|migrate|cleanup}" >&2
        echo "" >&2
        echo "  create   Create all new labels (idempotent)" >&2
        echo "  migrate  Move issues/PRs from old → new labels" >&2
        echo "  cleanup  Delete old labels" >&2
        exit 1
        ;;
esac
