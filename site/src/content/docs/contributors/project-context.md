---
title: Project Context Integration
description: Integration order and behavior for mine proj open
---

`mine proj` persists project registry in SQLite and per-project settings in `projects.toml`.

## Context Order

When `mine proj open <name>` runs:

1. Resolve project path from the registry (`projects` table).
2. Update `last_accessed` and `proj.current` in the key-value state store.
3. Shift previous current value to `proj.previous` for `pp` shell switching.
4. Return shell-consumable path (`--print-path`) for parent-shell `cd`.
5. Allow integrations to consume stored config (`tmux_layout`, `env_file`, `ssh_*`).

## Shell Contract

The shell helpers from `mine shell init` implement cwd switching:

- `p [name]` uses `mine proj open ... --print-path` then `cd`.
- `pp` uses `mine proj open --previous --print-path` then `cd`.

This avoids unsafe attempts to mutate parent-shell cwd from the CLI subprocess.

## Integration Notes

- `mine tmux`: `tmux_layout` is persisted per project and can be read by tmux workflows.
- `mine env`: `env_file` stores project-local env profile intent.
- `mine ssh`: `ssh_host` and `ssh_tunnel` define project defaults consumable by SSH commands.
