---
title: mine todo
description: Fast task management with priorities, due dates, and tags
---

Fast task management with priorities, due dates, and tags.

## Interactive TUI

When run in a terminal, `mine todo` launches a full-screen interactive browser:

```bash
mine todo         # interactive TUI (TTY) or plain list (piped)
mine todo --all   # include completed todos in the view
mine t            # alias
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
| `q` / `Esc` | Quit |

### Non-interactive (script-friendly)

When stdout is piped or not a TTY, `mine todo` prints the plain text list:

```bash
mine todo | grep "high"    # plain output for scripting
```

## List Todos

## Add a Todo

```bash
mine todo add "build the feature"
mine todo add "fix bug" -p high -d tomorrow
mine todo add "write docs" -p low -d 2026-03-01 --tags "docs,v0.2"
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
# Add a critical task due tomorrow
mine todo add "deploy to prod" -p crit -d tomorrow

# List all todos including completed
mine todo --all

# Mark task #3 as done
mine todo done 3

# Edit the title of task #2
mine todo edit 2 "updated task name"
```
