#!/usr/bin/env bash
# Syncs CHANGELOG.md from the repo root into the Starlight content directory
# with frontmatter prepended. Run automatically via prebuild/predev npm scripts.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SITE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SITE_DIR/.." && pwd)"

SOURCE="$REPO_ROOT/CHANGELOG.md"
TARGET="$SITE_DIR/src/content/docs/changelog.md"

if [ ! -f "$SOURCE" ]; then
  echo "Warning: $SOURCE not found, skipping changelog sync" >&2
  exit 0
fi

cat > "$TARGET" <<'FRONTMATTER'
---
title: Changelog
description: Release history and notable changes
---

FRONTMATTER

# Strip the "# Changelog" H1 header since Starlight renders the title from frontmatter.
# Keep everything else.
sed '1{/^# Changelog$/d;}' "$SOURCE" >> "$TARGET"

echo "Synced changelog to $TARGET"
