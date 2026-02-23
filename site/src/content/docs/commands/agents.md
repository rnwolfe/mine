---
title: mine agents
description: Manage coding agent configurations from a canonical store
---

Manage your coding agent configurations with a single canonical store of
instructions, rules, and skills — synced across Claude Code, Codex, Gemini CLI,
and OpenCode.

## Initialize

```bash
mine agents init
```

Creates the canonical agents store at `~/.local/share/mine/agents/` with a
full directory scaffold and a starter `instructions/AGENTS.md`.

## Detect Installed Agents

```bash
mine agents detect
```

Scans the system for installed coding agents and persists results to the
manifest. Detection checks both the agent binary in PATH and the agent's
config directory. Re-running is idempotent — the manifest is updated in place.

**Detected agents:**

| Agent | Binary | Config Directory |
|-------|--------|-----------------|
| Claude Code | `claude` | `~/.claude/` |
| Codex | `codex` | `~/.codex/` |
| Gemini CLI | `gemini` | `~/.gemini/` |
| OpenCode | `opencode` | `~/.config/opencode/` |

## Status

```bash
mine agents
```

Shows the agents store location, registered agent count, and link count.

## Store Layout

After `mine agents init`, the store contains:

```
~/.local/share/mine/agents/
├── .git/
├── .mine-agents          # manifest: detected agents, link mappings
├── instructions/
│   └── AGENTS.md         # shared agent instructions
├── skills/
├── commands/
├── agents/
├── settings/
├── mcp/
└── rules/
```

## Error Table

| Error | Cause | Fix |
|-------|-------|-----|
| `git init: exec: "git": not found` | git not in PATH | Install git |
| `reading manifest: parsing manifest` | Corrupt `.mine-agents` file | Remove and re-run `mine agents init` |
