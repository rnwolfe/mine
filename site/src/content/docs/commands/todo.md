---
title: mine todo
description: Task management with priorities, due dates, and tags
---

Fast task management with priorities, due dates, and tags.

## List todos

```bash
mine todo         # show open todos
mine todo --all   # include completed
mine t            # alias
```

## Add a todo

```bash
mine todo add "build the feature"
mine todo add "fix bug" -p high -d tomorrow
mine todo add "write docs" -p low -d 2026-03-01 --tags "docs,v0.2"
```

**Priorities**: `low` (or `l`), `med` (default), `high` (or `h`), `crit` (or `c`, `!`)

**Due dates**: `today`, `tomorrow` (or `tom`), `next-week` (or `nw`), `next-month` (or `nm`), or `YYYY-MM-DD`

## Complete a todo

```bash
mine todo done 1     # mark #1 as done
mine todo do 1       # alias
mine todo x 1        # alias
```

## Delete a todo

```bash
mine todo rm 1       # delete #1
mine todo remove 1   # alias
mine todo delete 1   # alias
```

## Edit a todo

```bash
mine todo edit 1 "new title"
```
