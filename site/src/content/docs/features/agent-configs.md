---
title: Agent Configuration Management
description: Version-controlled canonical store for your AI coding agent configs
---

Keep your AI agent configurations in sync and version-controlled. `mine agents` manages a canonical store for shared instructions, settings, and config files — with full git-backed version history.

## Key Capabilities

- **Canonical store** — single source of truth for all agent configs at `~/.local/share/mine/agents/`
- **Git-backed versioning** — commit snapshots and restore any file to any previous version
- **Structured layout** — scaffolded directories for instructions, skills, settings, MCP configs, and more
- **Restore to previous versions** — roll back any file in the store with a single command
- **Copy-mode link re-sync** — restoring a file automatically re-copies to linked agent directories

## Quick Example

```bash
# Initialize the canonical store
mine agents init

# Edit shared instructions
$EDITOR ~/.local/share/mine/agents/instructions/AGENTS.md

# Snapshot the current state
mine agents commit -m "initial agent setup"

# See version history
mine agents log

# Restore a file to a previous version
mine agents restore instructions/AGENTS.md --version abc1234
```

## How It Works

The canonical store lives at `~/.local/share/mine/agents/` with this layout:

```
agents/
├── .git/
├── .mine-agents          # manifest: tracked agents + link mappings
├── instructions/
│   └── AGENTS.md         # shared instructions for all agents
├── skills/
├── commands/
├── agents/
├── settings/
├── mcp/
└── rules/
```

`mine agents commit` stages all changes in the store and creates a git commit. Each commit is a restorable snapshot.

## Version Control

```bash
# Commit with a custom message
mine agents commit -m "add MCP server config"

# Commit with auto-generated message
mine agents commit

# View full history
mine agents log

# View history for a specific file
mine agents log instructions/AGENTS.md

# Restore to latest committed version
mine agents restore instructions/AGENTS.md

# Restore to a specific version
mine agents restore settings/claude.json --version abc1234
```

## Symlink vs Copy Mode

When files are distributed to agent directories via `mine agents link` (coming soon), they can be linked in two modes:

- **Symlink mode** — the agent directory contains a symlink to the canonical store file. Restoring the canonical file is instantly visible to the agent.
- **Copy mode** — the agent directory contains a copy. Restoring the canonical file automatically re-copies to all copy-mode targets.
