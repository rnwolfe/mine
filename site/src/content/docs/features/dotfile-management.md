---
title: Dotfile Management
description: Track, diff, and version your config files with git-backed storage
---

Keep your dotfiles safe without a separate dotfile manager. `mine stash` tracks config files in a git-backed local store, lets you diff changes, and keeps everything versioned.

## Key Capabilities

- **Track any file** — point at a config file and it's copied into the stash
- **Diff changes** — see which tracked files have been modified since last commit
- **Git-backed** — stash directory is a git repo, so you get full version history
- **List tracked files** — see all files you're managing with their source paths
- **XDG-compliant** — stash lives at `~/.local/share/mine/stash/`

## Quick Example

```bash
# Initialize the stash
mine stash init

# Track your shell and editor configs
mine stash track ~/.zshrc
mine stash track ~/.config/nvim/init.lua

# Check what's changed
mine stash diff
```

## How It Works

Run `mine stash init` once to create the stash directory. Then `mine stash track <file>` copies a file into the stash and starts tracking it. When you want to see what's changed, `mine stash diff` compares tracked files against their stash copies.

Since the stash directory is git-backed, you get commit history for free. Use `mine stash list` to see everything you're tracking.

## Learn More

See the [command reference](/commands/stash/) for all subcommands and detailed usage.
