---
title: Task Management
description: Fast task management with priorities, due dates, tags, and an interactive TUI
---

Stay on top of your work with a task system built for the terminal. `mine todo` gives you priorities, due dates, tags, and an interactive fuzzy-search TUI — all stored locally in SQLite, no cloud account needed.

## Key Capabilities

- **Four priority levels** — low, med, high, and crit — with natural-language shortcuts (`-p h`, `-p !`)
- **Due dates with shortcuts** — `tomorrow`, `next-week`, `next-month`, or explicit `YYYY-MM-DD`
- **Tags** — organize tasks with comma-separated labels (`--tags "docs,v0.2"`)
- **Interactive TUI** — full-screen fuzzy-search browser when running in a terminal
- **Script-friendly** — plain text output when piped, so it works in shell scripts and CI

## Quick Example

```bash
# Add a high-priority task due tomorrow
mine todo add "deploy staging" -p high -d tomorrow

# Browse tasks interactively
mine todo

# Mark task #3 as done
mine todo done 3
```

## How It Works

Run `mine todo` in a terminal and you get a full-screen picker — scroll, filter, toggle tasks done. Need to add something? Press `a` and type. Need to find a task? Press `/` and fuzzy-search. When you pipe the output (e.g., `mine todo | grep high`), it automatically switches to plain text.

Tasks are stored in a local SQLite database. No sync, no accounts, no latency. Everything responds instantly.

## Learn More

See the [command reference](/commands/todo/) for all subcommands, flags, and detailed usage.
