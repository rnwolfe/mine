---
title: mine dig
description: Pomodoro-style focus timer with streak tracking
---

Pomodoro-style focus timer with streak tracking and deep work stats.

## Start a Session

```bash
mine dig          # 25 minutes (default)
mine dig 45m      # 45 minutes
mine dig 1h       # 1 hour
```

Press `Ctrl+C` to end early. Sessions over 5 minutes still count toward your streak.

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
