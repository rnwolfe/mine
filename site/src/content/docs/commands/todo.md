---
title: mine todo
description: Fast task management with priorities, due dates, scheduling buckets, tags, and project scoping
---

Fast task management with priorities, due dates, scheduling buckets, tags, and project scoping.

## Interactive TUI

When run in a terminal, `mine todo` launches a full-screen interactive browser:

```bash
mine todo              # interactive TUI (TTY) or plain list (piped)
mine todo --done       # include completed todos in the view
mine todo --all        # show tasks from all projects + global
mine todo --someday    # include someday (hidden) tasks
mine todo --project p  # scope to a named project
mine t                 # alias
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `x` / `Space` / `Enter` | Toggle done / undone |
| `a` | Add new todo (type title, Enter to save) |
| `d` | Delete selected todo |
| `s` | Cycle schedule bucket (today → soon → later → someday) |
| `/` | Filter todos (fuzzy search) |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Esc` | Clear active filter (no-op if no filter) |
| `q` / `Ctrl+C` | Quit |

### Non-interactive (script-friendly)

When stdout is piped or not a TTY, `mine todo` prints the plain text list:

```bash
mine todo | grep "today"   # plain output for scripting
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--done` | | false | Include completed todos in the view |
| `--all` | `-a` | false | Show tasks from all projects and global |
| `--someday` | | false | Include someday tasks (hidden by default) |
| `--project` | | | Scope to a named project regardless of cwd |

> **Breaking change**: `--all/-a` now means "cross-project view" (was "show done"). Use `--done` to see completed tasks.

## Project Scoping

Tasks are automatically associated with the current project based on your working directory:

- **Inside a registered project** — `mine todo` shows that project's tasks plus global tasks
- **Outside any project** — shows only global tasks (no project binding)
- **`--project <name>`** — explicitly scope to any registered project; errors if not found
- **`--all`** — show tasks across all projects and global (project name shown as `@name` annotation)

> **Dashboard behavior**: `mine` (the dashboard) also uses cwd-based project resolution to show your todo count. This means the dashboard reflects the project containing your current directory, not the project explicitly opened via `mine proj open`. If you're outside any registered project, the dashboard shows global task counts.

```bash
# Auto-scope from cwd (inside /projects/myapp which is registered)
mine todo add "fix the bug"        # assigned to myapp project

# Explicit project assignment
mine todo add "update docs" --project myapp

# View all tasks with project annotations
mine todo --all
```

## Add a Todo

```bash
mine todo add "build the feature"
mine todo add "fix bug" -p high -d tomorrow
mine todo add "write docs" -p low -d 2026-03-01 --tags "docs,v0.2"
mine todo add "project task" --project myapp
mine todo add "tackle now" --schedule today
mine todo add "someday idea" --schedule someday
mine todo add "refactor auth" --note "context: auth is a mess, see issue #42"
```

### Priorities

- `low` (or `l`)
- `med` (default)
- `high` (or `h`)
- `crit` (or `c`, `!`)

### Due Dates

- `today`
- `tomorrow` (or `tom`)
- `next-week` (or `nw`)
- `next-month` (or `nm`)
- `YYYY-MM-DD` (explicit date)

### Schedule Buckets

Schedule buckets represent *when you intend to work on* a task (not when it's due):

- `today` (or `t`) — tackle it today; shown in bold gold
- `soon` (or `s`) — coming up, within a few days; shown in amber
- `later` (or `l`) — on the radar, not urgent; default
- `someday` (or `sd`) — aspirational; **hidden from default view**

```bash
mine todo add "urgent fix" --schedule today
mine todo add "maybe one day" --schedule sd
```

### Initial Note (Body)

Set an initial body/context when creating a task:

```bash
mine todo add "refactor auth" --note "context: current auth is a mess, see issue #42"
```

The body is shown in `mine todo show` output. Use it to capture why you're creating the task, links, or initial context.

## Schedule a Todo

Change the scheduling bucket for an existing task:

```bash
mine todo schedule 5 today     # set task #5 to today
mine todo schedule 3 soon      # set task #3 to soon
mine todo schedule 7 someday   # hide task #7 in someday
mine todo schedule 2 l         # short alias for later
mine todo schedule 1 sd        # short alias for someday
```

Short aliases: `t`=today, `s`=soon, `l`=later, `sd`=someday

Someday tasks are hidden from `mine todo` output by default. Use `mine todo --someday` to see them.

## What's Next? (Urgency Sort)

`mine todo next` surfaces the highest-urgency open task — the answer to "what should I work on?"

```bash
mine todo next        # show the single most urgent task
mine todo next 3      # show the top 3 most urgent tasks
```

Urgency is scored based on:

| Factor | Weight |
|--------|--------|
| Overdue | +100 |
| Schedule: today | +50 |
| Schedule: soon | +20 |
| Schedule: later | +5 |
| Priority: crit | +40 |
| Priority: high | +30 |
| Priority: med | +20 |
| Priority: low | +10 |
| Age (days, capped at 30) | +1/day |
| Current project boost | +10 |

- **Someday tasks are always excluded** from `next` results.
- When no open tasks exist, a friendly "all clear" message is shown.
- Output includes: title, priority, schedule, due date (if set), project, tags, age.

### Configurable Weights

Override defaults via `[todo.urgency]` in `~/.config/mine/config.toml`:

```toml
[todo.urgency]
overdue = 100
schedule_today = 50
schedule_soon = 20
schedule_later = 5
priority_crit = 40
priority_high = 30
priority_med = 20
priority_low = 10
age_cap = 30
project_boost = 10
```

Any unset field uses the default. This section is entirely optional.

The urgency sort is also the default sort order for `mine todo` list output.

## Add a Note to a Todo

Append a timestamped annotation to an existing task:

```bash
mine todo note 5 "tried approach X, failed — see PR #42"
mine todo note 5 "pairing with Sarah tomorrow on this"
```

Notes are stored with a timestamp and displayed chronologically in `mine todo show`. Use them to capture context, failed approaches, blockers, or links.

## Show Full Task Detail

Display a task's full detail card including body, all notes, and metadata:

```bash
mine todo show 5
```

Output includes: title, ID, priority, schedule, due date, project, tags, created/updated timestamps, body (if set), and all notes in chronological order. The notes section is omitted when there are no notes.

## Complete a Todo

```bash
mine todo done 1     # mark #1 as done
mine todo do 1       # alias
mine todo x 1        # alias
```

## Delete a Todo

```bash
mine todo rm 1       # delete #1
mine todo remove 1   # alias
mine todo delete 1   # alias
```

## Edit a Todo

```bash
mine todo edit 1 "new title"
```

## Examples

```bash
# Add a critical task due tomorrow, auto-scoped to current project
mine todo add "deploy to prod" -p crit -d tomorrow

# Add a task to tackle today
mine todo add "review PR" --schedule today

# Park an idea for someday
mine todo add "learn Rust" --schedule someday

# List completed todos too
mine todo --done

# List tasks across all projects
mine todo --all

# Show tasks including someday bucket
mine todo --someday

# Scope to a specific named project
mine todo --project myapp

# Set schedule on existing task
mine todo schedule 5 today

# Mark task #3 as done
mine todo done 3

# Edit the title of task #2
mine todo edit 2 "updated task name"
```

## Error Table

| Error | Cause | Fix |
|-------|-------|-----|
| `project "x" not found in registry` | `--project` name doesn't match any registered project | Run `mine proj list` to see valid project names |
| `"x" is not a valid todo ID` | Non-numeric ID passed to done/rm/edit/schedule/note/show | Use `mine todo` to see valid IDs |
| `invalid schedule "x"` | Unknown schedule bucket passed to `--schedule` or `schedule` subcommand | Use: `today` (t), `soon` (s), `later` (l), `someday` (sd) |
| `todo #N not found` | Note or show command references a non-existent task ID | Use `mine todo` to see valid IDs |
