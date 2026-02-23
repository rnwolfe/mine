---
title: mine agents
description: Manage and sync coding agent configurations
---

Manage a canonical store for your coding agent configurations. Distribute shared instructions, skills, and settings to all your agents, and sync across machines with a git remote.

## Initialize

```bash
mine agents init
```

Creates the canonical store at `~/.local/share/mine/agents/` with a scaffolded directory structure, initializes a git repository, and creates an initial commit. Safe to run multiple times — idempotent.

## Sync

### Configure a Remote

```bash
# Set the remote URL
mine agents sync remote git@github.com:you/agent-configs.git

# Show the current remote
mine agents sync remote
```

### Push to Remote

```bash
mine agents sync push
```

Pushes the canonical store's current branch to the configured remote.

### Pull from Remote

```bash
mine agents sync pull
```

Pulls from the configured remote with rebase. After pulling:
- **Copy-mode links** are automatically re-copied to their target agent directories
- **Symlink-mode links** require no action — they already point to the updated canonical store

If a rebase conflict occurs, an error message includes the path to the canonical store for manual resolution.

## Status

Running `mine agents` with no subcommand shows an overview of the store: location, number of commits, remote URL, and number of active links.

## Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `no version history yet` | Store not initialized (no git repo) | Run `mine agents init` first |
| `no commits yet` | Git repo exists but has no commits | Run `mine agents init` to create the initial commit |
| `no remote configured` | Push/pull without a remote | Run `mine agents sync remote <url>` |
| `pull failed — resolve conflicts` | Rebase conflict or uncommitted local changes | Resolve conflicts, or run `git stash` first if you have local changes |
