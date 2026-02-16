---
title: mine init
description: Guided first-time setup
---

Guided first-time setup. Creates config and data directories.

## Usage

```bash
mine init
```

## What It Does

1. Auto-detects your name from `~/.gitconfig`
2. Creates config at `~/.config/mine/config.toml`
3. Creates database at `~/.local/share/mine/mine.db`
4. Detects your shell for completion setup

## Example

```bash
$ mine init
⛏ Setting up mine...

What's your name? (detected from git: Ryan Wolfe)
> Ryan Wolfe

✓ Config created at ~/.config/mine/config.toml
✓ Database created at ~/.local/share/mine/mine.db

All set! Run `mine` to see your dashboard.

tip: Run `mine shell completions` to set up tab completions.
```
