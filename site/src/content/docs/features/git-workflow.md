---
title: Git Workflow
description: Git accelerator with branch picker, sweep, WIP commits, PR creation, and changelog generation
---

Speed up your daily git workflow without replacing git. `mine git` adds a fuzzy branch picker, one-command cleanup, WIP shortcuts, PR creation, changelog generation, and shell aliases — all shelling out to the real `git` binary.

## Key Capabilities

- **Branch picker** — fuzzy-searchable branch switcher with TUI
- **Sweep** — delete merged branches and prune stale remote refs in one command
- **WIP/undo** — save work-in-progress with `wip`, undo last commit with `undo`
- **PR creation** — auto-detects base branch, generates title and body from commits
- **Changelog** — generates Markdown changelogs from conventional commits
- **Aliases** — installs opinionated git aliases (`git co`, `git st`, `git lg`, etc.)

## Quick Example

```bash
# Switch branches interactively
mine git

# Clean up after a merge
mine git sweep

# Save and restore work in progress
mine git wip
mine git unwip

# Open a PR with auto-generated title
mine git pr
```

## How It Works

The bare `mine git` command opens a fuzzy branch picker — type to filter, Enter to switch. For day-to-day work, `mine git wip` stages everything and commits with "wip", and `mine git unwip` reverses it when you're ready to write a real commit. After merging, `mine git sweep` cleans up merged branches so you don't accumulate stale refs.

For releases, `mine git changelog --from v1.0.0` groups conventional commits into Features, Bug Fixes, and other categories. The shell functions (`gc`, `gp`, `gpl`, `gsw`) added by `mine shell init` fill in the gaps for common one-liners.

## Learn More

See the [command reference](/commands/git/) for all subcommands, flags, and detailed usage.
