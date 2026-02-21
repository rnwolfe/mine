---
title: Plugins
description: Extend mine with a sandboxed plugin ecosystem
---

Extend mine with third-party plugins that hook into any command, register custom subcommands, and run in a permission sandbox. Plugins are standalone binaries with a TOML manifest -- build them in any language, share them via GitHub.

## Key Capabilities

- **Sandboxed permissions** -- plugins declare what they need (network, filesystem, env vars) and you review it before installing
- **Hook into any command** -- four-stage pipeline (prevalidate, preexec, postexec, notify) with wildcard pattern matching
- **Custom commands** -- plugins can register their own subcommands under `mine <plugin> <command>`
- **GitHub discovery** -- search for community plugins directly from the CLI
- **Any language** -- shell scripts, Python, Go, Rust -- anything that reads JSON from stdin

## Quick Example

```bash
# Search GitHub for plugins
mine plugin search obsidian

# Install from a local directory
mine plugin install ./my-plugin

# List installed plugins
mine plugin list

# Show detailed info about a plugin
mine plugin info todo-stats
```

## How It Works

### The Plugin Pipeline

Plugins integrate through the same four-stage hook pipeline as [user-local hooks](/features/hooks/):

1. **prevalidate** -- modify arguments before validation
2. **preexec** -- modify validated context before execution
3. **postexec** -- modify results after execution
4. **notify** -- fire-and-forget side effects (logging, syncing, webhooks)

Plugin hooks and user-local hooks coexist in the same pipeline. Transform hooks chain alphabetically; notify hooks run in parallel.

### Manifest-Declared Permissions

Every plugin includes a `mine-plugin.toml` manifest that declares exactly what system resources it needs. mine displays these permissions at install time so you can review them before granting access.

| Permission | What it grants |
|------------|---------------|
| `network` | Outbound network access |
| `filesystem` | Read/write to specific paths |
| `store` | Read/write mine's SQLite database |
| `config_read` | Read mine's configuration (exposes `MINE_CONFIG_DIR`, `MINE_DATA_DIR`) |
| `config_write` | Write to mine's configuration |
| `env_vars` | Access to specific environment variables |

Plugins run with a minimal environment -- only `PATH` and `HOME` are always available. Everything else must be declared in the manifest and approved at install time.

### Installation Flow

When you run `mine plugin install`, mine:

1. Reads the `mine-plugin.toml` manifest from the source directory
2. Validates the manifest (required fields, stage/mode pairing, kebab-case name)
3. Displays the plugin's name, version, description, and requested permissions
4. Prompts for confirmation
5. Copies the plugin to `~/.local/share/mine/plugins/<name>/`
6. Registers hooks and commands in the plugin registry

## Learn More

- [Plugin command reference](/commands/plugin/) -- all subcommands, flags, and error handling
- [Building Plugins](/contributors/building-plugins/) -- how to create your own plugins
- [Plugin Protocol](/contributors/plugin-protocol/) -- the JSON communication contract
