---
title: mine agents
description: Manage coding agent configurations from a canonical store
---

Manage your coding agent configurations from a single canonical store. Keep instructions, rules, and skills in one place — then link them to each agent you use.

## Initialize the Store

```bash
mine agents init
```

Creates the canonical agents store at `~/.local/share/mine/agents/` with the following directory structure:

```
agents/
├── .git/
├── .mine-agents
├── instructions/
│   └── AGENTS.md    # starter instructions file
├── skills/
├── commands/
├── agents/
├── settings/
├── mcp/
└── rules/
```

Running `mine agents init` twice is safe — it is fully idempotent.

## Show Status

```bash
mine agents
```

Shows the store location, registered agents, and active link count.

## Subcommands

| Subcommand | Description |
|-----------|-------------|
| `mine agents init` | Create canonical store with starter directory structure |
