---
title: mine dig
description: Pomodoro-style focus timer with streak tracking
---

Pomodoro-style focus timer with streak tracking and deep work stats.

## Start a Session

```bash
mine dig          # 25 minutes (default), full-screen TUI in a terminal
mine dig 45m      # 45 minutes
mine dig 1h       # 1 hour
mine dig --simple # simple inline progress bar (no full-screen)
```

Press `q` or `Ctrl+C` to end early. Sessions over 5 minutes still count toward your streak.

## Full-Screen Mode

When run in a terminal, `mine dig` launches a full-screen focus timer that adapts to your terminal size, showing:

- A large countdown timer
- A progress bar proportional to time elapsed
- Elapsed and remaining time

### Keyboard Shortcuts (full-screen mode)

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` / `Esc` | End session early |

### Simple Mode

Use `--simple` to keep the original inline progress output (useful for tmux status bars or logging):

```bash
mine dig --simple
mine dig 45m --simple
```

### Non-interactive (piped output)

When stdout is piped or not a TTY, `mine dig` automatically uses simple mode:

```bash
mine dig | tee focus.log   # plain output for scripting
```

## View Stats

```bash
mine dig stats
```

Shows:
- Current streak (consecutive days with at least one session)
- Longest streak
- Total deep work time

## Examples

```bash
# Start a standard 25-minute session
mine dig

# Start a 90-minute deep work session
mine dig 90m

# View your focus stats
mine dig stats
```

## Tips

- Sessions under 5 minutes don't count toward streaks
- End early with `Ctrl+C` if you need to stop
- Check your stats to track productivity over time
