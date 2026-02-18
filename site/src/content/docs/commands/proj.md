---
title: mine proj
description: Project switcher and context manager
---

Register projects, switch quickly, and persist per-project context.

## Commands

| Command | Description |
|---------|-------------|
| `mine proj` | Fuzzy picker for registered projects (TTY) |
| `mine proj add [path]` | Register current directory or explicit path |
| `mine proj rm <name>` | Remove a registered project |
| `mine proj list` | List projects with path, last access, and branch |
| `mine proj open <name>` | Mark project active and emit context |
| `mine proj open --previous` | Switch to the previously active project |
| `mine proj scan [dir] --depth 3` | Discover git repos and register them |
| `mine proj config [key] [value]` | Read or set per-project settings |

## Examples

```bash
# Add current directory as a project
mine proj add .

# Interactive picker
mine proj

# Open by name
mine proj open mine

# Discover repos under ~/dev (3 levels deep by default)
mine proj scan ~/dev

# Save project-level SSH default
mine proj config ssh_host prod-box --project mine
```

## Project Config Keys

- `default_branch`
- `env_file`
- `tmux_layout`
- `ssh_host`
- `ssh_tunnel`
