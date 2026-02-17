#!/bin/bash
set -e

# Restore internal docs that should not be deleted
mkdir -p docs/internal/specs docs/plans docs/examples/plugins

git show HEAD:docs/internal/DECISIONS.md > docs/internal/DECISIONS.md
git show HEAD:docs/internal/STATUS.md > docs/internal/STATUS.md
git show HEAD:docs/internal/VISION.md > docs/internal/VISION.md
git show HEAD:docs/internal/specs/plugin-manifest.md > docs/internal/specs/plugin-manifest.md
git show HEAD:docs/plans/2026-02-15-issue-workflow-design.md > docs/plans/2026-02-15-issue-workflow-design.md
git show HEAD:docs/plans/2026-02-15-issue-workflow-implementation.md > docs/plans/2026-02-15-issue-workflow-implementation.md
git show HEAD:docs/examples/plugins/mine-plugin-obsidian.toml > docs/examples/plugins/mine-plugin-obsidian.toml
git show HEAD:docs/examples/plugins/mine-plugin-slack.toml > docs/examples/plugins/mine-plugin-slack.toml

echo "Internal docs restored"
