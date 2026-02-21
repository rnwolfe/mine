---
title: mine stash
description: Track, diff, and manage your dotfiles
---

Track, diff, and manage your dotfiles with git-backed versioning.

## Initialize

```bash
mine stash init
```

Creates a stash directory at `~/.local/share/mine/stash/`.

## Track a File

```bash
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
mine stash track ~/.config/starship.toml
```

Copies the file into the stash directory and tracks it for changes.

## List Tracked Files

```bash
mine stash list
```

Shows all currently tracked files with their source paths.

## Check for Changes

```bash
mine stash diff
```

Shows which tracked files have been modified since last commit.

## Commit Changes

```bash
mine stash commit "save dotfiles after brew update"
```

Records the current state of all tracked files as a git commit in the stash repo.

## Restore a File

```bash
mine stash restore ~/.zshrc
mine stash restore ~/.zshrc --version HEAD~1
mine stash restore ~/.zshrc --force
```

Restores a tracked file from the stash back to its source location.

| Flag | Short | Description |
|------|-------|-------------|
| `--version` | | Git ref to restore from (default: latest commit) |
| `--force` | `-f` | Override the restored file's permissions with the stash-recorded permissions (captured at track/commit time). Without this flag, the file's existing permissions are preserved. |

Without `--force`, the restored file keeps the current source file's permissions (or defaults to `0644` if the source file doesn't exist yet). Use `--force` when you want to restore both the content *and* the permissions exactly as they were when last committed.

## Sync with Remote

```bash
mine stash sync set-remote git@github.com:you/dotfiles.git
mine stash sync push
mine stash sync pull
```

Backs up and restores the stash repo from a remote git repository.

## Examples

```bash
# Initialize stash
mine stash init

# Track important config files
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
mine stash track ~/.config/nvim/init.lua

# Check what's changed
mine stash diff

# List all tracked files
mine stash list
```
