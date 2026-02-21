---
title: mine tips
description: Discover what mine can do with actionable daily tips
---

Discover what mine can do. Shows a single daily-rotating tip, or lists all tips with `--all`.

## Usage

```bash
mine tips          # Show today's tip
mine tips --all    # List all tips
```

## Daily Tip

`mine tips` (no flags) prints one tip for the current day. The tip is deterministic — the same tip appears all day, then changes at midnight. Run it whenever you open a terminal to gradually discover mine's full feature set.

```
  tip: `mine proj add .` to register this directory as a project.

  Run `mine tips --all` to see all tips.
```

## List All Tips

```bash
mine tips --all
```

Prints the full tip pool — 25+ actionable one-liners covering all major features: todos, projects, AI, stash, vault, env profiles, tmux, hooks, plugins, and more.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-a` | List all tips instead of showing today's tip |

## Examples

```bash
# See today's tip
mine tips

# Browse the full tip library
mine tips --all
```

## Tips

- The dashboard also shows a daily rotating tip when your todo list is empty.
- Tips are curated to be actionable — each one is a runnable command, not generic advice.
