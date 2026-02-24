---
title: Task Management
description: Fast task management with priorities, due dates, tags, project scoping, and an interactive TUI
---

Stay on top of your work with a task system built for the terminal. `mine todo` gives you priorities, due dates, tags, project binding, and an interactive fuzzy-search TUI — all stored locally in SQLite, no cloud account needed.

## Key Capabilities

- **Four priority levels** — low, med, high, and crit — with natural-language shortcuts (`-p h`, `-p !`)
- **Due dates with shortcuts** — `tomorrow`, `next-week`, `next-month`, or explicit `YYYY-MM-DD`
- **Tags** — organize tasks with comma-separated labels (`--tags "docs,v0.2"`)
- **Project scoping** — tasks auto-bind to your current project based on cwd; global tasks work everywhere
- **Cross-project view** — `--all` shows every task across all projects plus global
- **Interactive TUI** — full-screen fuzzy-search browser when running in a terminal
- **Script-friendly** — plain text output when piped, so it works in shell scripts and CI

## Quick Example

```bash
# Add a high-priority task due tomorrow (auto-scoped to current project if in one)
mine todo add "deploy staging" -p high -d tomorrow

# Browse tasks for the current project
mine todo

# Show tasks across all projects
mine todo --all

# Assign a task to a specific project regardless of cwd
mine todo add "write changelog" --project myapp

# Mark task #3 as done
mine todo done 3
```

## How It Works

Run `mine todo` in a terminal and you get a full-screen picker — scroll, filter, toggle tasks done. Need to add something? Press `a` and type. Need to find a task? Press `/` and fuzzy-search. When you pipe the output (e.g., `mine todo | grep high`), it automatically switches to plain text.

Tasks are stored in a local SQLite database. No sync, no accounts, no latency. Everything responds instantly.

### Project Binding

When you run `mine todo` inside a registered project directory (one added via `mine proj add`), tasks are automatically scoped to that project. Running outside any project shows only global tasks (those not tied to any project).

Use `--all` to see everything across all projects, or `--project <name>` to explicitly scope to a named project.

## Learn More

See the [command reference](/commands/todo/) for all subcommands, flags, and detailed usage.
