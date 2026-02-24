---
title: mine todo
description: Fast task management with priorities, due dates, tags, and project scoping
---

Fast task management with priorities, due dates, tags, and project scoping.

## Interactive TUI

When run in a terminal, `mine todo` launches a full-screen interactive browser:

```bash
mine todo              # interactive TUI (TTY) or plain list (piped)
mine todo --done       # include completed todos in the view
mine todo --all        # show tasks from all projects + global
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
| `/` | Filter todos (fuzzy search) |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `Esc` | Clear active filter (no-op if no filter) |
| `q` / `Ctrl+C` | Quit |

### Non-interactive (script-friendly)

When stdout is piped or not a TTY, `mine todo` prints the plain text list:

```bash
mine todo | grep "high"    # plain output for scripting
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--done` | | false | Include completed todos in the view |
| `--all` | `-a` | false | Show tasks from all projects and global |
| `--project` | | | Scope to a named project regardless of cwd |

> **Breaking change**: `--all/-a` now means "cross-project view" (was "show done"). Use `--done` to see completed tasks.

## Project Scoping

Tasks are automatically associated with the current project based on your working directory:

- **Inside a registered project** — `mine todo` shows that project's tasks plus global tasks
- **Outside any project** — shows only global tasks (no project binding)
- **`--project <name>`** — explicitly scope to any registered project; errors if not found
- **`--all`** — show tasks across all projects and global (project name shown as `@name` annotation)

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

# List completed todos too
mine todo --done

# List tasks across all projects
mine todo --all

# Scope to a specific named project
mine todo --project myapp

# Mark task #3 as done
mine todo done 3

# Edit the title of task #2
mine todo edit 2 "updated task name"
```

## Error Table

| Error | Cause | Fix |
|-------|-------|-----|
| `project "x" not found in registry` | `--project` name doesn't match any registered project | Run `mine proj list` to see valid project names |
| `"x" is not a valid todo ID` | Non-numeric ID passed to done/rm/edit | Use `mine todo` to see valid IDs |
