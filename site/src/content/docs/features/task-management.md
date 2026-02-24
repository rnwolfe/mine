---
title: Task Management
description: Fast task management with GTD-style scheduling buckets, priorities, due dates, tags, project scoping, and an interactive TUI
---

Stay on top of your work with a task system built for the terminal. `mine todo` gives you GTD-style scheduling buckets, priorities, due dates, tags, project binding, and an interactive fuzzy-search TUI — all stored locally in SQLite, no cloud account needed.

## Key Capabilities

- **Scheduling buckets** — `today`, `soon`, `later`, `someday` separate *when you intend to work* from *when it's due*
- **Someday parking lot** — someday tasks are hidden from default views; surface them with `--someday`
- **Four priority levels** — low, med, high, and crit — with natural-language shortcuts (`-p h`, `-p !`)
- **Due dates with shortcuts** — `tomorrow`, `next-week`, `next-month`, or explicit `YYYY-MM-DD`
- **Tags** — organize tasks with comma-separated labels (`--tags "docs,v0.2"`)
- **Project scoping** — tasks auto-bind to your current project based on cwd; global tasks work everywhere
- **Cross-project view** — `--all` shows every task across all projects plus global
- **Interactive TUI** — full-screen fuzzy-search browser when running in a terminal; press `s` to cycle schedule
- **Script-friendly** — plain text output when piped, so it works in shell scripts and CI

## Quick Example

```bash
# Add a high-priority task due tomorrow, auto-scoped to current project
mine todo add "deploy staging" -p high -d tomorrow

# Add a task to tackle today (scheduling intent)
mine todo add "review PR" --schedule today

# Park an aspirational idea in the someday bucket
mine todo add "learn Rust" --schedule someday

# Browse tasks (someday hidden by default)
mine todo

# Show tasks including the someday bucket
mine todo --someday

# Set the schedule on an existing task
mine todo schedule 5 today

# Show tasks across all projects
mine todo --all

# Mark task #3 as done
mine todo done 3
```

## How It Works

Run `mine todo` in a terminal and you get a full-screen picker — scroll, filter, toggle tasks done. Press `a` to add, `/` to fuzzy-search, `s` to cycle the schedule bucket of the selected task. When you pipe the output (e.g., `mine todo | grep today`), it automatically switches to plain text.

Tasks are stored in a local SQLite database. No sync, no accounts, no latency. Everything responds instantly.

### Scheduling Buckets

Four buckets represent when you *intend* to work on a task:

| Bucket | Intent | Default list |
|--------|--------|-------------|
| `today` | Tackle it today | ✓ Shown |
| `soon` | Coming up, within a few days | ✓ Shown |
| `later` | On the radar, not urgent (default) | ✓ Shown |
| `someday` | Aspirational, no commitment | Hidden |

Someday tasks are intentionally hidden from `mine todo` to keep your daily view uncluttered. Use `mine todo --someday` to see them, or `mine todo schedule <id> someday` to park a task.

### Project Binding

When you run `mine todo` inside a registered project directory (one added via `mine proj add`), tasks are automatically scoped to that project. Running outside any project shows only global tasks (those not tied to any project).

Use `--all` to see everything across all projects, or `--project <name>` to explicitly scope to a named project.

## Learn More

See the [command reference](/commands/todo/) for all subcommands, flags, and detailed usage.
