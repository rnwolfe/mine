---
title: mine grow
description: Career growth tracking — learning goals, activity log, streaks, and skill levels
---

Track your learning journey with goals, activity logs, streaks, and self-assessed skills.

## Usage

```
mine grow [subcommand]
```

Running `mine grow` with no subcommand shows the dashboard.

## Subcommands

### `mine grow` (dashboard)

Show an at-a-glance summary: current streak, active goal count, and top skills.

```bash
mine grow
```

---

### `mine grow goal add <title>`

Add a learning or career goal.

```bash
mine grow goal add "Learn Rust" --deadline 2026-06-01 --target 50 --unit hrs
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--deadline` | — | Target completion date (`YYYY-MM-DD`) |
| `--target N` | `0` | Numeric target (e.g. `50` for 50 hrs) |
| `--unit <str>` | `hrs` | Unit label (e.g. `hrs`, `sessions`) |

---

### `mine grow goal list`

List all active goals with progress bars.

```bash
mine grow goal list
```

---

### `mine grow goal done <id>`

Mark a goal as complete.

```bash
mine grow goal done 1
```

---

### `mine grow log [note]`

Log a learning activity. The note is optional.

```bash
mine grow log "Read The Rust Book ch. 3" --minutes 45 --goal 1 --skill Rust
mine grow log --minutes 30 --skill "System Design"
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--minutes N` | `0` | Time spent in minutes |
| `--goal <id>` | — | Link to a goal ID (updates goal progress) |
| `--skill <name>` | — | Skill tag for this activity |

After logging, the current streak is shown. If `--goal` is provided, goal progress updates
automatically by summing all minutes logged against that goal.

---

### `mine grow streak`

Show the current and longest learning streak (consecutive days with ≥1 activity).

```bash
mine grow streak
```

Streaks are not broken if you logged yesterday but haven't logged today yet.

---

### `mine grow skills`

Display all self-assessed skills grouped by category, with dot-notation levels.

```bash
mine grow skills
```

Example output:
```
  programming
    Rust                     ●●●○○  (3/5)
    Go                       ●●●●○  (4/5)
```

---

### `mine grow skills set <name> <1-5>`

Set the self-assessed level for a skill. Creates the skill if it doesn't exist.

```bash
mine grow skills set Rust 3
mine grow skills set "System Design" 2 --category architecture
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--category <str>` | `general` | Category to group the skill under |

Level scale:

| Level | Meaning |
|-------|---------|
| 1 | Aware — know it exists |
| 2 | Beginner — can do basics |
| 3 | Intermediate — productive |
| 4 | Advanced — confident |
| 5 | Expert — deep knowledge |

---

### `mine grow review`

Weekly and monthly summary: streak, activity counts, total minutes, goal progress,
and recent activities.

```bash
mine grow review
```

## Configuration

```toml
[grow]
default_minutes = 30   # Default minutes when --minutes is not specified
```

## Hook Events

| Event | Trigger |
|-------|---------|
| `grow.log` | After an activity is logged |
| `grow.goal.add` | After a goal is created |
| `grow.goal.done` | After a goal is marked complete |
