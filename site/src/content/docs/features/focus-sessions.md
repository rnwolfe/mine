---
title: Focus Sessions
description: Pomodoro-style focus timer with streak tracking, task targeting, and focus time display
---

Block out distractions and get into deep work. `mine dig` is a Pomodoro-style focus timer with a full-screen TUI, streak tracking, lifetime stats, and task integration — all from your terminal.

## Key Capabilities

- **Customizable duration** — default 25 minutes, or specify any length (`45m`, `1h`, `90m`)
- **Full-screen TUI** — large countdown, progress bar, elapsed/remaining time
- **Simple mode** — inline progress bar for tmux status bars or logging (`--simple`)
- **Streak tracking** — consecutive days with at least one session (minimum 5 minutes counts)
- **Lifetime stats** — total deep work time, current streak, longest streak, and session count
- **Task targeting** — link a session to a specific task with `--todo <id>`
- **Task picker** — when inside a project, a picker offers open tasks before the session starts
- **Completion prompt** — after a linked session ends, prompts "Mark #N done? (y/n)"
- **Focus time in todo list** — accumulated time shows inline as `[25m]` in `mine todo` output

## Quick Example

```bash
# Start a standard 25-minute session
mine dig

# Focus on a specific task
mine dig --todo 12

# Deep work for 90 minutes, linked to task #3
mine dig 90m --todo 3

# Check your focus stats
mine dig stats
```

## Task Integration

Link focus sessions to tasks to close the capture → focus → complete loop:

```bash
mine dig --todo 12
# Displays "Focusing on: Refactor auth module" in the timer
# After session: "Mark #12 done? (y/n)"
```

When you run `mine dig` inside a registered project without `--todo`, a task picker appears automatically so you can select a task before the timer starts. Press `Esc` to skip and start an untargeted session.

## Focus Time in Task List

Accumulated focus time from linked sessions appears inline in `mine todo` list output:

```
    #1   🟡 [today]   Refactor auth module  [1h 25m]
    #2   🔴 [soon]    Fix login bug
```

Focus time is only shown when > 0 — no `[0m]` clutter for untouched tasks.

## How It Works

Run `mine dig` and a full-screen timer takes over your terminal. A progress bar fills as time elapses. Press `q` or `Ctrl+C` to end early — sessions over 5 minutes still count toward your streak and focus time. When the session completes, it's logged automatically.

Use `mine dig stats` to see your current streak, longest streak, total deep work hours, and session count. It's a simple feedback loop: focus, track, complete.

## Learn More

See the [command reference](/commands/dig/) for all options and detailed usage.
