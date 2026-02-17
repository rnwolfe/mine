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
