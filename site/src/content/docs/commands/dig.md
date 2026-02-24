---
title: mine dig
description: Pomodoro-style focus timer with streak tracking and task integration
---

Pomodoro-style focus timer with streak tracking, deep work stats, and task linking.

## Start a Session

```bash
mine dig          # 25 minutes (default), full-screen TUI in a terminal
mine dig 45m      # 45 minutes
mine dig 1h       # 1 hour
mine dig --simple # simple inline progress bar (no full-screen)
```

Press `q` or `Ctrl+C` to end early (full-screen mode only). Sessions over 5 minutes still count toward your streak.

## Link a Session to a Task

```bash
mine dig --todo 12    # start a session targeting task #12
mine dig --todo 999   # error: todo #999 not found
```

When you provide `--todo <id>`, the session is linked to that task:
- The task title is displayed in the focus timer during the session
- After the session ends (≥ 5 min), you are prompted: **Mark #12 done? (y/n)**
- Answering `y` marks the task complete immediately

## Task Picker (inside a project)

When you run `mine dig` inside a registered project without `--todo`, you are offered a task picker showing the project's open tasks:

```bash
cd ~/projects/myapp
mine dig   # → shows task picker for myapp's open tasks
```

Press `Esc` to skip the picker and start an untargeted session. If no open tasks exist or you are outside any registered project, the session starts immediately without a picker.

## Full-Screen Mode

When run in a terminal, `mine dig` launches a full-screen focus timer that adapts to your terminal size, showing:

- A large countdown timer
- A progress bar proportional to time elapsed
- Elapsed and remaining time
- The linked task title (when `--todo` or picker is used)

### Keyboard Shortcuts (full-screen mode)

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | End session early |

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

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--todo <id>` | 0 (unset) | Link session to a task by ID |
| `--simple` | false | Use simple inline progress bar instead of full-screen TUI |

## View Stats

```bash
mine dig stats
```

Shows:
- Current streak (consecutive days with at least one session)
- Longest streak
- Total deep work time
- Total session count
- Number of distinct tasks focused on (when sessions are task-linked)

## Examples

```bash
# Start a standard 25-minute session
mine dig

# Start a 90-minute deep work session
mine dig 90m

# Link a session to task #5
mine dig --todo 5

# View your focus stats
mine dig stats
```

## Tips

- Sessions under 5 minutes don't count toward streaks or focus time
- End early with `Ctrl+C` if you need to stop — sessions ≥ 5 min still count
- Focus time per task appears in `mine todo` list output as `[25m]`, `[1h 30m]`, etc.
- Task picker only appears when inside a registered project with open tasks
