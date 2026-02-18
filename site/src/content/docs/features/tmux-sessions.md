---
title: Tmux Sessions
description: Fuzzy session picker, layout persistence, and session management for tmux
---

Manage tmux sessions without memorizing session names. `mine tmux` gives you a fuzzy picker for sessions, one-command create/attach/kill, and the ability to save and restore window layouts.

## Key Capabilities

- **Fuzzy session picker** — interactive searchable list of running sessions
- **Auto-naming** — creates sessions named after the current directory when no name is given
- **Layout persistence** — save your window/pane layout and restore it later
- **Fuzzy attach** — `mine tmux attach proj` fuzzy-matches to the right session
- **Script-friendly** — plain list output when piped, interactive picker in a terminal

## Quick Example

```bash
# Pick a session interactively
mine tmux

# Create a new session (auto-named from current directory)
mine tmux new

# Save your current layout
mine tmux layout save dev-setup

# Restore it later
mine tmux layout load dev-setup
```

## How It Works

The bare `mine tmux` command opens a fuzzy picker showing all running sessions — select one and you're attached. `mine tmux new` creates a session named after your current directory (or pass an explicit name). `mine tmux attach` supports fuzzy matching, so `mine tmux attach proj` will find a session named "my-project".

Layouts capture your window and pane arrangement. Run `mine tmux layout save dev-setup` inside a tmux session and the layout is saved to `~/.config/mine/`. Later, `mine tmux layout load dev-setup` restores it. Use `mine tmux layout ls` to see all saved layouts with window counts and names.

## Learn More

See the [command reference](/commands/tmux/) for all subcommands and detailed usage.
