---
title: Building Plugins
description: How to build, test, and publish mine plugins
---

A mine plugin is a standalone binary paired with a TOML manifest. Plugins can hook into any mine command, register custom subcommands, and respond to lifecycle events. You can build them in any language -- shell, Python, Go, Rust -- as long as the binary reads JSON from stdin and writes JSON to stdout.

## Quick Start

Build a working plugin in 5 minutes using the `todo-stats` example as a starting point.

### 1. Create the directory

```bash
mkdir my-plugin && cd my-plugin
```

### 2. Write the manifest

Create `mine-plugin.toml`:

```toml
[plugin]
name = "todo-stats"
version = "0.1.0"
description = "Track todo completion stats and show a summary"
author = "your-name"
license = "MIT"
min_mine_version = "0.2.0"
protocol_version = "1.0.0"

[[hooks]]
command = "todo.done"
stage = "notify"
mode = "notify"

[[commands]]
name = "summary"
description = "Show todo completion stats"
```

### 3. Write the binary

Create `mine-plugin-todo-stats` (the default entrypoint name):

```bash
#!/bin/sh
set -e

STATS_FILE="${HOME}/.local/share/mine/todo-stats.log"
INPUT=$(cat)
TYPE=$(echo "$INPUT" | sed -n 's/.*"type":"\([^"]*\)".*/\1/p')
COMMAND=$(echo "$INPUT" | sed -n 's/.*"command":"\([^"]*\)".*/\1/p')
EVENT=$(echo "$INPUT" | sed -n 's/.*"event":"\([^"]*\)".*/\1/p')

case "$TYPE" in
  hook)
    mkdir -p "$(dirname "$STATS_FILE")"
    echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) todo.done" >> "$STATS_FILE"
    ;;
  command)
    if [ "$COMMAND" = "summary" ]; then
      if [ ! -f "$STATS_FILE" ]; then
        echo "No completions recorded yet."
        exit 0
      fi
      COUNT=$(wc -l < "$STATS_FILE" | tr -d ' ')
      echo "Todo completions: $COUNT"
    else
      echo "Unknown command: $COMMAND" >&2
      exit 1
    fi
    ;;
  lifecycle)
    if [ "$EVENT" = "health" ]; then
      echo '{"status": "ok"}'
    fi
    ;;
esac
```

Make it executable:

```bash
chmod +x mine-plugin-todo-stats
```

### 4. Install and test

```bash
mine plugin install .
mine todo-stats summary
```

## The Manifest

The `mine-plugin.toml` manifest declares everything about your plugin. Here is a complete annotated example:

### `[plugin]` section

```toml
[plugin]
name = "my-plugin"          # Required. Kebab-case (lowercase, digits, hyphens).
version = "0.1.0"           # Required. Semver.
description = "What it does" # Required. Short, one-line description.
author = "your-name"        # Required.
license = "MIT"             # Optional. SPDX identifier.
min_mine_version = "0.2.0"  # Optional. Minimum mine version required.
protocol_version = "1.0.0"  # Required. Must match mine's supported protocol.
entrypoint = "my-binary"    # Optional. Defaults to "mine-plugin-<name>".
```

The `name` field must be kebab-case: lowercase letters, digits, and hyphens only. The name determines the default entrypoint binary name (`mine-plugin-<name>`) and the install directory.

### `[[hooks]]` section

```toml
[[hooks]]
command = "todo.done"   # Command pattern (supports wildcards)
stage = "notify"        # One of: prevalidate, preexec, postexec, notify
mode = "notify"         # One of: transform, notify
timeout = "15s"         # Optional. Overrides default timeout.
```

You can declare multiple `[[hooks]]` entries. Each one registers the plugin for a specific command pattern and stage.

**Command patterns** use dot-separated segments with wildcard support:
- `todo.add` -- matches only `todo add`
- `todo.*` -- matches all todo subcommands
- `*` -- matches every command

**Stage/mode pairing rules** (enforced by manifest validation):
- `notify` stage requires `notify` mode
- `prevalidate`, `preexec`, and `postexec` stages require `transform` mode

Violating these rules causes a validation error at install time.

### `[[commands]]` section

```toml
[[commands]]
name = "sync"                              # Subcommand name
description = "Sync todos to external service" # Shown in help text
args = "[--vault <path>]"                  # Optional. Usage hint for arguments.
```

Commands are invoked as `mine <plugin-name> <command>`. For example, a plugin named `obsidian-sync` with a `sync` command is called via `mine obsidian-sync sync`.

### `[permissions]` section

```toml
[permissions]
network = true                        # Outbound network access
filesystem = ["~/.obsidian", "~/notes"] # Read/write to specific paths
store = true                          # Read/write mine's SQLite database
config_read = true                    # Read mine config (exposes MINE_CONFIG_DIR, MINE_DATA_DIR)
config_write = false                  # Write to mine config
env_vars = ["OBSIDIAN_VAULT"]         # Access to specific environment variables
```

All permissions default to `false` or empty. Only declare what you actually need -- users see every permission at install time.

## The Protocol

Plugins communicate with mine via JSON over stdin/stdout. There are three invocation types: `hook`, `command`, and `lifecycle`. Each invocation includes a `type` field and a `protocol_version` field.

### Hook invocation

mine sends the hook context on stdin:

```json
{
  "protocol_version": "1.0.0",
  "type": "hook",
  "stage": "preexec",
  "mode": "transform",
  "context": {
    "command": "todo.add",
    "args": ["buy milk"],
    "flags": {"priority": "high"},
    "result": null,
    "timestamp": "2026-01-15T10:30:00Z"
  }
}
```

For transform hooks, write modified context to stdout:

```json
{
  "status": "ok",
  "context": {
    "command": "todo.add",
    "args": ["buy milk"],
    "flags": {"priority": "high", "tags": "auto-tagged"},
    "result": null,
    "timestamp": "2026-01-15T10:30:00Z"
  }
}
```

For notify hooks, stdout is ignored. Perform side effects and exit.

### Command invocation

```json
{
  "protocol_version": "1.0.0",
  "type": "command",
  "command": "sync",
  "args": ["--vault", "notes"]
}
```

Write raw output to stdout (not JSON). This is displayed directly to the user. Flag parsing is the plugin's responsibility.

### Lifecycle invocation

```json
{
  "protocol_version": "1.0.0",
  "type": "lifecycle",
  "event": "health"
}
```

Supported events: `init` (startup), `shutdown` (exit), `health` (status check). For `health`, respond with `{"status": "ok"}`.

For the full protocol specification, see the [Plugin Protocol](/contributors/plugin-protocol/) reference.

## Walkthrough Examples

The `docs/examples/plugins/` directory in the mine repository contains complete working examples.

### Shell: todo-stats

A minimal shell plugin that logs todo completions and provides a summary command. No permissions required.

- Manifest: `docs/examples/plugins/todo-stats/mine-plugin.toml`
- Binary: `docs/examples/plugins/todo-stats/mine-plugin-todo-stats`

### Python: webhook

A Python plugin that validates todo input (transform hook) and sends webhook notifications (notify hook). Requires network access and a `WEBHOOK_URL` environment variable.

- Manifest: `docs/examples/plugins/webhook/mine-plugin.toml`
- Binary: `docs/examples/plugins/webhook/mine-plugin-webhook`

### Go: tag-enforcer

A Go plugin that enforces tagging policies on todos. Demonstrates prevalidate and postexec hooks, wildcard pattern matching, and protocol version checking.

- Manifest: `docs/examples/plugins/tag-enforcer/mine-plugin.toml`
- Binary: `docs/examples/plugins/tag-enforcer/main.go` (build with `go build -o mine-plugin-tag-enforcer .`)

For Go plugins, build a static binary and set the `entrypoint` field in the manifest to match the binary name. The same JSON protocol applies — use `encoding/json` to read from stdin and write to stdout.

## Permissions

Plugins run in a restricted environment. mine builds a minimal set of environment variables for each plugin subprocess:

| Always available | Conditionally available |
|-----------------|------------------------|
| `PATH` | Declared `env_vars` (filtered from host environment) |
| `HOME` | `MINE_CONFIG_DIR` (if `config_read` is true) |
| | `MINE_DATA_DIR` (if `config_read` is true) |

Everything else is stripped. A plugin that declares `env_vars = ["WEBHOOK_URL"]` only receives `PATH`, `HOME`, and `WEBHOOK_URL` (if set in the host environment). If a declared env var is not set, mine logs a warning.

All declared permissions are displayed during `mine plugin install` so the user can make an informed decision before granting access.

## Publishing

To make your plugin discoverable via `mine plugin search`:

1. **Name your repository** `mine-plugin-<name>` (e.g., `mine-plugin-obsidian`)
2. **Add the GitHub topic** `mine-plugin` to your repository
3. **Include a `mine-plugin.toml`** at the repository root
4. **Include the binary or build instructions** in the README

Users discover your plugin with:

```bash
mine plugin search obsidian
mine plugin search --tag logging
```

The search uses the GitHub search API to find repositories matching the `mine-plugin-*` naming convention, optionally filtered by topic.

## Testing

### Manual testing during development

Install directly from your working directory:

```bash
mine plugin install .
```

This reads the manifest and binary from the current directory. Re-run after changes to update the installed plugin.

### Testing hooks

Use `mine hook test` to dry-run your plugin's hook with sample input:

```bash
mine hook test ~/.local/share/mine/plugins/my-plugin/mine-plugin-my-plugin
```

This sends sample JSON on stdin and displays the output, without actually executing any command.

### Verifying the manifest

The install command validates the manifest before proceeding. Common validation errors:

- Missing required fields (`name`, `version`, `description`, `author`, `protocol_version`)
- Name not in kebab-case format
- Invalid stage/mode pairing (e.g., `stage = "preexec"` with `mode = "notify"`)
- Missing hook or command fields

If validation fails, mine reports the specific error so you can fix the manifest.

## See Also

- [Plugin Protocol](/contributors/plugin-protocol/) -- full JSON communication specification
- [Plugin command reference](/commands/plugin/) -- all CLI subcommands and flags
- [Hooks feature overview](/features/hooks/) -- user-local hooks (no plugin system needed)
