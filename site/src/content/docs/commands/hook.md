---
title: mine hook
description: User-local hooks for customizing mine commands
---

Drop scripts into `~/.config/mine/hooks/` to hook into any mine command — no plugin system needed. Scripts are auto-discovered and run as part of the command pipeline.

## How Hooks Work

Hooks intercept commands at four stages:

| Stage | Mode | When it runs |
|-------|------|-------------|
| `prevalidate` | transform | Before input validation |
| `preexec` | transform | Before the command executes |
| `postexec` | transform | After the command executes |
| `notify` | notify | Fire-and-forget, after everything else |

**Transform** hooks receive JSON on stdin and return modified JSON on stdout. They chain in alphabetical order — each hook's output becomes the next hook's input.

**Notify** hooks receive JSON on stdin but their output is ignored. They run in parallel and never block the command.

## Filename Convention

Scripts follow a naming convention: `<command-pattern>.<stage>.<ext>`

```
~/.config/mine/hooks/
├── todo.add.preexec.sh         # Runs before todo add
├── todo.done.notify.py         # Notified after todo done
├── todo.*.postexec.sh          # Runs after any todo subcommand
└── *.notify.sh                 # Notified on every command
```

- The command pattern uses dot-separated segments (`todo.add`, `todo.*`, `*`)
- Wildcard `*` matches any characters (Go `filepath.Match` rules) — a bare `*` matches all commands
- Scripts must be executable (`chmod +x`)
- Any language works — bash, python, ruby, compiled binaries

## JSON Protocol

Hooks receive a JSON context on stdin:

```json
{
  "command": "todo.add",
  "args": ["buy milk"],
  "flags": {"priority": "high"},
  "timestamp": "2026-01-15T10:30:00Z"
}
```

The `result` field is included after command execution (`postexec` and `notify` stages) and omitted in earlier stages.

Transform hooks write modified JSON to stdout. Notify hooks can ignore output.

## List Hooks

```bash
mine hook              # show all discovered hooks
mine hook list         # same thing
```

## Create a Hook

```bash
mine hook create todo.add preexec      # hook for todo add
mine hook create "todo.*" notify       # hook for all todo subcommands
mine hook create "*" postexec          # hook for every command
```

This scaffolds a starter script with inline comments explaining the protocol.

## Test a Hook

```bash
mine hook test ~/.config/mine/hooks/todo.add.preexec.sh
```

Dry-runs the hook with sample input and shows the output.

## Timeouts

- **Transform hooks**: 5 seconds (blocks the command)
- **Notify hooks**: 30 seconds (runs in background)

If a hook exceeds its timeout, it's killed and an error is reported.

## Examples

```bash
# Create a hook that runs before adding todos
mine hook create todo.add preexec
# Edit it to add your logic
$EDITOR ~/.config/mine/hooks/todo.add.preexec.sh

# Create a notify hook for all commands
mine hook create "*" notify
# Test it
mine hook test ~/.config/mine/hooks/*.notify.sh

# List all active hooks
mine hook list
```

See `docs/examples/hooks/` in the repository for complete example scripts.
