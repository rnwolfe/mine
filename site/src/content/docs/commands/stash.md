---
title: mine stash
description: Track, diff, and manage your dotfiles
---

Git-backed dotfile versioning. Track, diff, and sync your configuration files.

## Initialize

```bash
mine stash init
```

Creates a stash directory at `~/.local/share/mine/stash/`.

## Track a file

```bash
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
mine stash track ~/.config/starship.toml
```

## List tracked files

```bash
mine stash list
```

## Check for changes

```bash
mine stash diff
```

Shows which tracked files have been modified since last stash.

## Commit changes

```bash
mine stash commit -m "update zsh config"
```

## View history

```bash
mine stash log
```

## Restore a file

```bash
mine stash restore ~/.zshrc
```

## Sync with remote

```bash
mine stash sync
```

Requires a git remote configured in `~/.local/share/mine/stash/.git/config`.
