---
title: Agent Management
description: Keep coding agent instructions and configs in one canonical store
---

Manage all your coding agent configurations from a single canonical store. `mine agents` tracks instructions, rules, skills, and settings for agents like Claude, Codex, and Gemini — in one git-backed directory you control.

## Key Capabilities

- **Canonical store** — one directory for all agent configurations at `~/.local/share/mine/agents/`
- **Structured layout** — directories for instructions, skills, commands, rules, settings, and MCP configs
- **Git-backed** — the store is a git repo, so every change is tracked
- **Starter template** — `init` creates an `AGENTS.md` with helpful comments to get you started
- **XDG-compliant** — follows `~/.local/share/mine/agents/` path convention

## Quick Example

```bash
# Initialize the agents store
mine agents init

# Edit your shared instructions
vim ~/.local/share/mine/agents/instructions/AGENTS.md

# Check store status
mine agents
```

## How It Works

Run `mine agents init` once to create the store. The store is a git repository so you can commit changes as you evolve your agent configurations. The `instructions/AGENTS.md` starter file is a good place to put shared context you want all agents to see.

## Learn More

See the [command reference](/commands/agents/) for all subcommands and detailed usage.
