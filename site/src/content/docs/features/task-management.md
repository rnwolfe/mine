---
title: Task Management
description: Fast task management with urgency-sorted "what next?", GTD-style scheduling buckets, priorities, due dates, tags, project scoping, and an interactive TUI
---

Stay on top of your work with a task system built for the terminal. `mine todo` gives you GTD-style scheduling buckets, priorities, due dates, tags, project binding, urgency-ranked output, and an interactive fuzzy-search TUI — all stored locally in SQLite, no cloud account needed.

## Key Capabilities

- **"What next?" urgency sort** — `mine todo next` answers the eternal question with a weighted score across overdue status, schedule, priority, age, and project context
- **Scheduling buckets** — `today`, `soon`, `later`, `someday` separate *when you intend to work* from *when it's due*
- **Someday parking lot** — someday tasks are hidden from default views; surface them with `--someday`
- **Four priority levels** — low, med, high, and crit — with natural-language shortcuts (`-p h`, `-p !`)
- **Due dates with shortcuts** — `tomorrow`, `next-week`, `next-month`, or explicit `YYYY-MM-DD`
- **Tags** — organize tasks with comma-separated labels (`--tags "docs,v0.2"`)
- **Project scoping** — tasks auto-bind to your current project based on cwd; global tasks work everywhere
- **Cross-project view** — `--all` shows every task across all projects plus global
- **Notes and annotations** — append timestamped notes to tasks with `mine todo note`; view full detail with `mine todo show`
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

# Browse tasks sorted by urgency (someday hidden by default)
mine todo

# Answer "what should I work on right now?"
mine todo next

# Show the top 3 most urgent tasks
mine todo next 3

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

### Urgency Scoring ("What Next?")

`mine todo next` computes a score for every open, non-someday task and surfaces the highest-scoring one:

| Factor | Points |
|--------|--------|
| Overdue (past due date) | +100 |
| Schedule: today | +50 |
| Schedule: soon | +20 |
| Schedule: later | +5 |
| Priority: crit | +40 |
| Priority: high | +30 |
| Priority: med | +20 |
| Priority: low | +10 |
| Age (1/day, capped at 30) | up to +30 |
| Current project match | +10 |

Overdue tasks always rank above non-overdue tasks. Someday tasks are excluded entirely. The urgency sort is also the default sort order for the regular `mine todo` list view.

Power users can tune the weights via `[todo.urgency]` in `~/.config/mine/config.toml`.

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

## Notes and Annotations

Capture context, failed approaches, and links directly on a task so you don't lose them.

```bash
# Add initial context when creating a task
mine todo add "refactor auth" --note "see issue #42 — current impl is fragile"

# Append a timestamped note to an existing task
mine todo note 5 "tried approach X, failed — see PR #77"
mine todo note 5 "pairing with Sarah tomorrow"

# View full task detail including all notes
mine todo show 5
```

`mine todo show` renders a detail card with the task's schedule, priority, due date, project, tags, body, and all notes in chronological order. The notes section is omitted when there are none.

## Completion Stats and Velocity

See how your productivity trends over time with `mine todo stats`:

```bash
# View completion streak, weekly count, monthly count, avg close time
mine todo stats

# Scope stats to a specific project
mine todo stats --project myapp
```

Output includes:
- **Streak** — consecutive days with at least one completion (longest ever shown in parentheses)
- **This week / This month** — completion counts with Monday-start weeks and calendar months
- **Avg close** — average days from task creation to completion (completed tasks only)
- **Focus time** — total accumulated focus time from linked `mine dig` sessions (when available)
- **By project** — per-project breakdown with open count, completed count, and average close time

When no completions exist, an encouraging prompt is shown rather than an error.

## Learn More

See the [command reference](/commands/todo/) for all subcommands, flags, and detailed usage.
