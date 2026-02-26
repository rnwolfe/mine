---
title: Dashboard
description: Interactive TUI dashboard showing todos, focus stats, and project context at a glance
---

Replace the static `mine` summary with a live, keyboard-driven TUI dashboard. Run `mine` (or `mine dash`) in a terminal to see your most urgent todos, focus streak, and current project — all in one view.

## Key Capabilities

- **At-a-glance overview** — todos, focus stats, and project context in one screen
- **Responsive layout** — two-column side-by-side at ≥120 cols; single-column stacked below that; minimal mode below 60 cols
- **Top-5 urgent tasks** — shows your most pressing todos sorted by urgency, with overdue markers
- **Focus stats panel** — current streak, this week's completions, and total accumulated focus time
- **Project panel** — current project name, git branch, and open todo count
- **Quick actions** — open the full todo TUI, start a dig session, or refresh data without leaving the dashboard
- **Keyboard-driven** — no mouse required; all navigation via single keystrokes
- **TTY-aware** — pipes produce plain text automatically; ANSI codes only in interactive terminals

## Quick Example

```bash
# Launch the TUI dashboard (default when in a terminal)
mine

# Explicit alias — same as typing mine with no args in a TTY
mine dash

# Print static text output (original behavior, useful for scripts)
mine --plain

# Pipe output always produces plain text
mine | cat
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `t` | Open the full interactive todo TUI |
| `d` | Start a 25-minute dig focus session |
| `r` | Refresh all panel data from the store |
| `q` | Quit the dashboard |
| `Ctrl+C` | Quit the dashboard |

## Layout

The dashboard adapts to your terminal width:

| Width | Layout |
|-------|--------|
| ≥ 120 cols | Two-column: todos on the left, focus stats + project on the right |
| 60–119 cols | Single-column stacked: todos, then focus stats, then project |
| < 60 cols | Minimal mode: key counts only, no decorations |

## Panels

### Todos Panel

Shows the top 5 most urgent open tasks sorted by the same urgency algorithm as `mine todo next`. Each row includes the task ID, priority icon, schedule bucket, and title. Overdue tasks are highlighted in red.

### Focus Panel

Displays your current completion streak (consecutive days with at least one completed task), this week's completion count, and total accumulated deep work time from `mine dig` sessions.

### Project Panel

Shows the current project (detected from your working directory), active git branch, and the count of open todos scoped to that project. Absent when you're not inside a registered project.

## How It Works

1. `mine` checks whether **stdout** is connected to a terminal (not stdin, so `mine | cat` correctly detects the pipe)
2. If yes and `mine init` has been run: launches the Bubbletea TUI dashboard
3. If stdout is piped/redirected, `--plain` flag is set, or `mine init` hasn't run: falls back to the original static text output
4. The `t` key pauses the dashboard and opens the full todo TUI; returning from it re-shows the dashboard
5. The `d` key runs a 25-minute focus session; the dashboard re-opens when the session ends
6. Data is fetched once on startup; `r` re-fetches without restarting
