---
title: Hooks & Plugins
description: Extend mine with user-local hooks and the plugin system
---

Customize and extend mine without forking it. User-local hooks let you intercept any command with a simple script. The plugin system lets you install third-party extensions that declare their required capabilities in a manifest.

## Key Capabilities

- **User-local hooks** — drop scripts into `~/.config/mine/hooks/` to intercept any command
- **Four hook stages** — prevalidate, preexec, postexec, and notify
- **Filename convention** — `<command-pattern>.<stage>.<ext>` (e.g., `todo.add.preexec.sh`)
- **Wildcard patterns** — `todo.*` matches all todo subcommands, `*` matches everything
- **Plugin system** — install third-party plugins that declare required environment variables, filesystem paths, and network access

## Quick Example

```bash
# Create a hook that runs before adding todos
mine hook create todo.add preexec

# List all active hooks
mine hook list

# Test a hook with sample input
mine hook test ~/.config/mine/hooks/todo.add.preexec.sh
```

## How It Works

### Hooks

Hooks are scripts that run at specific points in a command's lifecycle. Drop an executable file into `~/.config/mine/hooks/` with the right name and it's picked up automatically. The filename convention is parsed right-to-left: extension, then stage, then command pattern.

**Transform hooks** (prevalidate, preexec, postexec) receive JSON on stdin and return modified JSON on stdout. They chain alphabetically — each hook's output feeds the next. **Notify hooks** receive JSON on stdin but run fire-and-forget in the background, never blocking the command.

Use `mine hook create` to scaffold a starter script with inline comments explaining the protocol, and `mine hook test` to dry-run it.

### Plugins

Plugins are standalone binaries invoked via JSON-over-stdin. They declare capabilities in a `mine-plugin.toml` manifest — including which environment variables, filesystem paths, and network access they require. mine displays these declared permissions at install time so you know what a plugin needs before enabling it. To install a plugin from GitHub, clone or download it first, then run `mine plugin install <path>`.

## Learn More

See the [hook command reference](/commands/hook/) for hook stages, the JSON protocol, and timeouts.
