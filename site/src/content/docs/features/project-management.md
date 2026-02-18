---
title: Project Management
description: Register projects, switch context fast, and persist per-project settings
---

Keep all your projects organized and switch between them instantly. `mine proj` maintains a registry of your git repos, tracks which one you're working in, and lets you jump between them from anywhere in your shell.

## Key Capabilities

- **Project registry** — register git repos by path with auto-detected names
- **Fuzzy picker** — interactive searchable list of registered projects
- **Fast switching** — `p <name>` jumps to any project; `pp` switches to the previous one
- **Context memory** — tracks current and previous project so `pp` always works
- **Repo discovery** — scan a directory tree and register all git repos at once
- **Per-project settings** — store SSH defaults, tmux layouts, env files per project

## Quick Example

```bash
# Register the current directory as a project
mine proj add .

# Pick a project interactively
mine proj

# Jump to a project by name (and cd into it)
p mine

# Switch back to the previous project
pp

# Discover and register all repos under ~/dev
mine proj scan ~/dev
```

## How It Works

`mine proj add` registers a directory in a local SQLite registry. The project name is derived from the directory basename. Once registered, `mine proj` opens a fuzzy picker to browse and switch projects — select one and the path is emitted. The `p` and `pp` shell functions (added by `mine shell init`) capture that path and `cd` into it, so you jump directories without any unsafe parent-shell mutation.

Every time you open a project, `mine proj` records it as the current project and shifts the previous current to a "previous" slot. That's what powers `pp` — it always takes you back to where you just were.

`mine proj scan ~/dev --depth 3` walks a directory tree and registers any git repo it finds, so you can onboard all your projects at once. `mine proj config` lets you attach per-project metadata — the `ssh_host`, `ssh_tunnel`, `tmux_layout`, and `env_file` keys are consumed by other `mine` commands when you're working in that project.

## Learn More

See the [command reference](/commands/proj/) for all subcommands, flags, and per-project config keys.
