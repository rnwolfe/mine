---
title: Career Growth Tracking
description: Track learning goals, build streaks, and self-assess skills with mine grow
---

Never lose track of your learning journey. `mine grow` is a low-friction activity log
that compounds into streaks, goal progress, and a self-assessed skill picture — all local,
all yours.

## Key Capabilities

- **Learning goals** — set goals with optional deadline and target (e.g. 50 hrs of Rust)
- **Activity log** — log time spent learning, tagged to a goal and/or skill
- **Streak tracking** — consecutive calendar days with at least one activity logged
- **Grace period** — logging yesterday but not yet today keeps your streak alive
- **Skill radar** — self-assess skill levels 1–5 rendered as `●●●○○`
- **Weekly review** — summary of activities, goal progress, and streak for the week/month
- **Dashboard** — at-a-glance view of streak, active goals, and top skills

## Quick Example

```bash
# Add a learning goal
mine grow goal add "Learn Rust" --deadline 2026-06-01 --target 50 --unit hrs

# Log a learning session
mine grow log "Read The Rust Book ch. 3" --minutes 45 --goal 1 --skill Rust

# Check your streak
mine grow streak

# Self-assess a skill
mine grow skills set Rust 3

# Weekly summary
mine grow review
```

## How It Works

### Goals

Goals have an optional target value (e.g. 50 hrs) and unit (e.g. `hrs`, `sessions`).
Progress is tracked automatically: every `mine grow log` that references a goal
accumulates its minutes toward `current_value`. You can mark goals complete with
`mine grow goal done <id>`.

### Activity Log

Each log entry captures a note, duration in minutes, an optional goal link, and a skill tag.
The `--minutes` flag defaults to the `grow.default_minutes` config value (if set) or 0.

### Streaks

A streak counts consecutive calendar days with at least one logged activity. If you
logged yesterday but haven't logged today yet, your streak remains active — this prevents
the streak from breaking at midnight before you've had a chance to log.

### Skill Levels

Skills are stored as a name, category, and integer level (1–5). The level is displayed as
filled/empty dots:

| Level | Display |
|-------|---------|
| 1     | `●○○○○` |
| 3     | `●●●○○` |
| 5     | `●●●●●` |

### Storage

All data is stored in the mine SQLite database. Three tables are created automatically:
`grow_goals`, `grow_activities`, `grow_skills`. No cloud sync — local first.

## Configuration

Add to `~/.config/mine/config.toml`:

```toml
[grow]
default_minutes = 30   # Default session length when --minutes is not set
```
