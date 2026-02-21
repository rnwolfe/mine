---
title: Hooks
description: Customize mine commands with user-local hook scripts
---

Customize any mine command by dropping a script into `~/.config/mine/hooks/`. No plugin system needed -- just an executable file with the right name and mine picks it up automatically.

## Key Capabilities

- **Zero config** -- drop a script into `~/.config/mine/hooks/` and it's active immediately
- **Four hook stages** -- prevalidate, preexec, postexec, and notify
- **Wildcard patterns** -- `todo.*` matches all todo subcommands, `*` matches everything
- **Any language** -- bash, python, ruby, Go binaries, anything executable
- **Transform or notify** -- modify command context or fire-and-forget side effects

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

### Filename Convention

Scripts follow a naming convention parsed right-to-left: `<command-pattern>.<stage>.<ext>`

```
~/.config/mine/hooks/
├── todo.add.preexec.sh         # Runs before todo add
├── todo.done.notify.py         # Notified after todo done
├── todo.*.postexec.sh          # Runs after any todo subcommand
└── *.notify.sh                 # Notified on every command
```

The command pattern uses dot-separated segments. Wildcard `*` matches any characters (Go `filepath.Match` rules) -- a bare `*` matches all commands. Scripts must be executable (`chmod +x`).

### Four Stages

Every mine command passes through four hook stages:

| Stage | Mode | When it runs |
|-------|------|-------------|
| `prevalidate` | transform | Before input validation |
| `preexec` | transform | Before the command executes |
| `postexec` | transform | After the command executes |
| `notify` | notify | Fire-and-forget, after everything else |

### Transform vs Notify

**Transform hooks** (prevalidate, preexec, postexec) receive JSON on stdin and return modified JSON on stdout. They chain in alphabetical order -- each hook's output becomes the next hook's input. Use these when you need to modify the command's arguments, flags, or results.

**Notify hooks** receive JSON on stdin but their output is ignored. They run in the background and never block the command. Use these for side effects like logging, syncing to external services, or sending notifications.

### Timeouts

- **Transform hooks**: 5 seconds (blocks the command pipeline)
- **Notify hooks**: 30 seconds (runs in background)

If a hook exceeds its timeout, mine kills the process and reports an error.

## CLI Commands

| Command | Description |
|---------|-------------|
| `mine hook list` | List all discovered hooks |
| `mine hook create <pattern> <stage>` | Scaffold a starter hook script |
| `mine hook test <path>` | Dry-run a hook with sample input |

## Learn More

See the [hook command reference](/commands/hook/) for the full JSON protocol, all subcommands, and detailed examples.
