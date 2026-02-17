---
title: mine todo
description: Fast task management with priorities, due dates, and tags
---

Fast task management with priorities, due dates, and tags.

## List Todos

```bash
mine todo         # show open todos
mine todo --all   # include completed
mine t            # alias
```

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
