---
title: Quick Start
description: Get up and running with mine in minutes
---

## Your First Commands

After [installation](/getting-started/installation), run the guided setup:

```bash
mine init
```

Then try these commands to get a feel for mine:

### View Your Dashboard

```bash
mine
```

You'll see your personal dashboard with todos, the date, and version info.

### Add Your First Todo

```bash
mine todo add "try out mine" -p high -d today
```

**Priorities**: `low` (or `l`), `med` (default), `high` (or `h`), `crit` (or `c`, `!`)

**Due dates**: `today`, `tomorrow` (or `tom`), `next-week` (or `nw`), or `YYYY-MM-DD`

### List Your Todos

```bash
mine todo
```

### Complete a Todo

```bash
mine todo done 1
```

### Start a Focus Session

```bash
mine dig          # 25m pomodoro (default)
mine dig 45m      # custom duration
```

Press `Ctrl+C` to end early. Sessions over 5 minutes still count toward your streak.

### Track Your Dotfiles

```bash
mine stash init
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
```

### Check for Changes

```bash
mine stash diff
```

### Bootstrap a Project

```bash
mine craft dev go       # Go project
mine craft dev node     # Node.js
mine craft git          # git init + .gitignore
```

### Set Up Shell Aliases

```bash
mine shell aliases
```

This prints recommended aliases you can add to your shell config:

```bash
alias m='mine'
alias mt='mine todo'
alias mta='mine todo add'
alias mtd='mine todo done'
alias md='mine dig'
alias mc='mine craft'
alias ms='mine stash'
```

## Configuration

Config lives at `~/.config/mine/config.toml`:

```toml
[user]
name = "Your Name"

[shell]
default_shell = "/usr/bin/zsh"
```

Edit it directly:

```bash
$EDITOR $(mine config path)
```

Or view it:

```bash
mine config
```

## Data Storage

- **Config**: `~/.config/mine/config.toml`
- **Database**: `~/.local/share/mine/mine.db` (SQLite)
- **Stash**: `~/.local/share/mine/stash/` (git repo)

## Next Steps

- [Explore all commands](/commands/todo)
- [Learn about the architecture](/contributing/architecture)
- [Write a plugin](/contributing/plugin-protocol)
