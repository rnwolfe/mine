---
title: mine dig
description: Pomodoro-style focus timer with streak tracking
---

Deep work focus timer. Build the habit of focused work with streak tracking.

## Start a session

```bash
mine dig          # 25 minutes (default)
mine dig 45m      # 45 minutes
mine dig 1h       # 1 hour
```

Press `Ctrl+C` to end early. Sessions over 5 minutes still count toward your streak.

## View stats

```bash
mine dig stats
```

Shows:
- Current streak (consecutive days with at least one session)
- Longest streak
- Total deep work time

## How it works

1. Start a session with `mine dig`
2. Focus on your work
3. The timer shows elapsed time and remaining time
4. Complete the session or end early (min 5 minutes)
5. Session is logged and your streak is updated

Streaks are preserved across days — complete at least one session per day to maintain your streak.
