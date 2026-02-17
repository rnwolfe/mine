# Issue #28: User-local hooks (~/.config/mine/hooks/)

## Status
READY

## Labels
enhancement, phase:2, agent-ready, in-progress, maestro

## Worktree
/home/rnwolfe/dev/mine-worktrees/issue-28

## Body
## Summary

Allow users to drop scripts into `~/.config/mine/hooks/` that hook into any mine command — no plugin system needed. This is the "escape hatch" for quick customization.

**Parent tracker:** #12
**Depends on:** #26 (hook execution pipeline)

## Scope

### Convention-based Hook Scripts

Scripts in `~/.config/mine/hooks/` are auto-discovered by filename convention:

```
~/.config/mine/hooks/
├── todo.add.preexec.sh         # Runs before todo add
├── todo.done.notify.py         # Notified after todo done
├── todo.*.postexec.sh          # Runs after any todo command
└── *.notify.sh                 # Notified on every command
```

**Filename format:** `<command-pattern>.<stage>.<ext>`

- Any executable file (sh, py, rb, compiled binary — anything with +x)
- Follows the same JSON stdin/stdout protocol as plugins
- Transform hooks chain: output of one becomes input of next (alphabetical order)
- Notify hooks are fire-and-forget

### Hook Management Commands

```
mine hook list                  # Show all registered hooks
mine hook test <file>           # Dry-run a hook with sample input
mine hook create <pattern>      # Scaffold a new hook script from template
```

### Templates

`mine hook create todo.add.preexec` generates a starter script:

```bash
#!/usr/bin/env bash
# mine hook: todo.add @ preexec (transform)
# Receives command context as JSON on stdin, return modified context on stdout.

input=$(cat)
# Modify and echo back:
echo "$input"
```

## Acceptance Criteria

- [ ] Scripts in `~/.config/mine/hooks/` are auto-discovered and registered
- [ ] Filename convention parsed correctly (command pattern, stage, extension)
- [ ] Scripts must be executable (+x) — warn on non-executable files
- [ ] Wildcard patterns work in filenames (`todo.*`, `*`)
- [ ] Transform hooks chain in alphabetical order
- [ ] Notify hooks run in parallel, fire-and-forget
- [ ] `mine hook list` shows all discovered hooks with status
- [ ] `mine hook test` dry-runs a hook with sample Context JSON
- [ ] `mine hook create` scaffolds a starter script with correct template
- [ ] Timeout on hook execution (default 5s for transform, 30s for notify)
- [ ] Clear error messages when hooks fail (path, exit code, stderr)
- [ ] Tests for discovery, filename parsing, execution, chaining, and timeouts

## Documentation

- [ ] `docs/hooks.md` — **user-facing** guide: how to write hooks, filename conventions, examples
- [ ] `docs/examples/hooks/` — 2-3 example hook scripts (e.g. auto-tag todos, Obsidian sync notify, Slack notify)
- [ ] Update `mine hook create` templates with inline comments explaining the protocol

## Agentic Knowledge Base

- [ ] Update `CLAUDE.md` (symlinked as `agents.md`) with any new architecture patterns, key files, domain packages, or lessons learned introduced by this work

## PR
- **Number:** 111
- **URL:** https://github.com/rnwolfe/mine/pull/111
- **Branch:** maestro/issue-28-user-local-hooks
