---
title: mine proj
description: Project registry and context switcher
---

Register projects, switch context fast, and persist per-project settings.

## Fuzzy Project Picker

```bash
mine proj       # interactive picker (TTY) or plain list (piped)
```

Opens a fuzzy-searchable list of registered projects. Select a project and press Enter to open it. Falls back to a plain list when stdout is not a TTY.

## Register a Project

```bash
mine proj add           # register current directory (name from directory basename)
mine proj add ~/dev/api # register an explicit path
mine proj add . --name myapi  # register with a custom name
```

Registers a directory in the project registry. The project name is auto-detected from the directory basename if not specified. Adding a project that is already registered returns an error.

## Remove a Project

```bash
mine proj rm myapi       # remove by name (prompts for confirmation)
mine proj rm myapi --yes # skip confirmation prompt
mine proj rm myapi -y    # short flag
```

Removes a project from the registry. Per-project settings stored in `projects.toml` are also cleaned up.

## List Projects

```bash
mine proj list
mine proj ls
```

Lists all registered projects with name, path, last accessed timestamp, and current git branch (best-effort).

## Open a Project

```bash
mine proj open myapi             # mark project active and print context
mine proj open myapi --print-path # print path only (used by shell helpers)
mine proj open --previous         # reopen the previously active project
```

Marks a project as the current project and updates the last-accessed timestamp. Records the previous current project so `pp` can switch back. With `--print-path`, only the project path is printed to stdout — this is how the `p` and `pp` shell functions perform `cd` in the parent shell.

## Discover Repos

```bash
mine proj scan ~/dev            # scan with default depth (3)
mine proj scan ~/dev --depth 4  # scan deeper
mine proj scan .                # scan current directory tree
```

Recursively walks a directory and registers any git repos found. Stops at repos (does not recurse into them) and respects the depth limit. Already-registered repos are silently skipped.

## Per-Project Config

```bash
mine proj config                             # list all settings for current project
mine proj config ssh_host                    # get a single setting
mine proj config ssh_host prod-box           # set a setting
mine proj config ssh_host prod-box -p myapi  # target a specific project
```

Reads or writes per-project settings stored in `~/.config/mine/projects.toml`. Settings are silently ignored if a project has no entry yet.

### Config Keys

| Key | Description |
|-----|-------------|
| `default_branch` | Default git branch name (e.g. `main`) |
| `env_file` | Path to a project-local env file |
| `tmux_layout` | Saved tmux layout name to load on open |
| `ssh_host` | Default SSH host alias for this project |
| `ssh_tunnel` | Default SSH tunnel spec for this project |

## Shell Helpers

```bash
p [name]   # fuzzy-pick or open a project and cd into it
pp         # switch to the previously active project and cd into it
```

The `p` and `pp` functions are installed by `mine shell init`. They call `mine proj open --print-path` and use the returned path to `cd` in the current shell process — avoiding any attempt to mutate a parent shell from a subprocess.

## Examples

```bash
# Register all repos under ~/dev
mine proj scan ~/dev

# Interactive picker
mine proj

# Jump to a project by name (via shell function)
p myapi

# Switch back to the previous project
pp

# Save a tmux layout for the project
mine proj config tmux_layout dev-setup

# Set an SSH default for the project
mine proj config ssh_host prod-box --project myapi
```
