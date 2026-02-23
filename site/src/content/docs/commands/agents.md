---
title: mine agents
description: Manage and version-control your AI agent configurations
---

Canonical store for AI agent configs — initialize, snapshot, and restore your agent setups with full git-backed version history.

## Initialize

```bash
mine agents init
```

Creates the canonical store at `~/.local/share/mine/agents/` with subdirectories for instructions, skills, commands, settings, MCP configs, and rules. Initializes a git repository for version tracking.

Safe to run multiple times — idempotent.

## Commit a Snapshot

```bash
mine agents commit
mine agents commit -m "add shared instructions"
```

Stages all changes in the canonical store and commits them. Initializes git versioning on first run if needed.

When `-m` is omitted, a timestamp-based message is generated automatically.

If nothing has changed since the last commit, prints a message and exits cleanly.

**Flags:**

| Flag | Description |
|------|-------------|
| `-m`, `--message` | Commit message (optional) |

## View History

```bash
mine agents log
mine agents log instructions/AGENTS.md
```

Shows commit history for the canonical store. Optionally filter to a specific file (relative path within the store).

Output format: `<short-hash> <message> (<age>)`

## Restore a File

```bash
mine agents restore <file>
mine agents restore <file> --version <hash>
```

Restores a file in the canonical store to a previous version. The `<file>` argument is a path relative to the canonical store (e.g., `instructions/AGENTS.md`).

- **Symlink-mode links** propagate automatically — the symlink already points to the canonical file.
- **Copy-mode links** are re-synced automatically — the restored content is re-copied to all linked targets.

If `--version` is omitted, restores from the latest commit (HEAD).

**Flags:**

| Flag | Description |
|------|-------------|
| `-v`, `--version` | Git commit hash to restore from (default: latest) |

## Status

```bash
mine agents
```

Shows the canonical store location, snapshot count, and the most recent commit.

## Error Reference

| Error | Cause | Fix |
|-------|-------|-----|
| `agents store not initialized` | `mine agents init` not run | Run `mine agents init` |
| `nothing to commit` | No changes since last commit | Make changes first |
| `no version history yet` | No commits in the store | Run `mine agents commit` |
| `version X not found for Y` | Hash does not exist or file not in that commit | Use `mine agents log <file>` to find valid hashes |
| `unsafe file path` | Path contains `..` or is absolute | Use relative paths like `instructions/AGENTS.md` |
