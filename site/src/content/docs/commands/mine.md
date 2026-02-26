---
title: mine (root command)
description: Default command — launches the TUI dashboard or prints a static summary
---

Running `mine` with no subcommand is the primary entry point. In a terminal it opens the interactive TUI dashboard; piped or with `--plain` it prints a static at-a-glance summary.

## Usage

```bash
mine [--plain]
mine dash
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--plain` | `false` | Print static text dashboard instead of launching the TUI |

## Subcommand: `mine dash`

`mine dash` is an explicit alias for the TUI dashboard. Useful for shell scripts or keybindings that want to open the dashboard directly, regardless of whether `mine` would choose the TUI or static output.

```bash
mine dash    # always opens the TUI (falls back to static if not a TTY)
```

## Behavior

| Condition | Result |
|-----------|--------|
| stdout is a TTY + `mine init` run | Full TUI dashboard |
| `--plain` set | Static text summary (original behavior) |
| stdout is piped / redirected | Static text summary (stdout TTY check fails) |
| `mine init` not yet run | Welcome screen with setup instructions |

## TUI Keyboard Shortcuts

When the dashboard TUI is open:

| Key | Action |
|-----|--------|
| `t` | Open the full interactive todo TUI |
| `d` | Start a 25-minute dig focus session |
| `r` | Refresh all panel data from the store |
| `q` / `Ctrl+C` | Quit |

## Examples

```bash
# Open TUI dashboard (default in a terminal after mine init)
mine

# Force static text output (e.g. for a shell prompt snippet)
mine --plain

# Explicit TUI alias
mine dash

# Script-friendly: always plain text
mine --plain | grep open
```

## Related

- [`mine todo`](/commands/todo) — full task management
- [`mine dig`](/commands/dig) — focus sessions
- [`mine proj`](/commands/proj) — project management
- [`mine init`](/commands/init) — first-time setup
