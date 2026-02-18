---
title: Focus Sessions
description: Pomodoro-style focus timer with streak tracking and deep work stats
---

Block out distractions and get into deep work. `mine dig` is a Pomodoro-style focus timer with a full-screen TUI, streak tracking, and lifetime stats — all from your terminal.

## Key Capabilities

- **Customizable duration** — default 25 minutes, or specify any length (`45m`, `1h`, `90m`)
- **Full-screen TUI** — large countdown, progress bar, elapsed/remaining time
- **Simple mode** — inline progress bar for tmux status bars or logging (`--simple`)
- **Streak tracking** — consecutive days with at least one session (minimum 5 minutes counts)
- **Lifetime stats** — total deep work time, current streak, and longest streak

## Quick Example

```bash
# Start a standard 25-minute session
mine dig

# Deep work for 90 minutes
mine dig 90m

# Check your focus stats
mine dig stats
```

## How It Works

Run `mine dig` and a full-screen timer takes over your terminal. A progress bar fills as time elapses. Press `q` or `Ctrl+C` to end early — sessions over 5 minutes still count toward your streak. When the session completes, it's logged automatically.

Use `mine dig stats` to see your current streak, longest streak, and total deep work hours. It's a simple feedback loop: focus, track, improve.

## Learn More

See the [command reference](/commands/dig/) for all options and detailed usage.
