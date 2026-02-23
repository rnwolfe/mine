---
title: Agent Config Management
description: One canonical store for all your coding agent configurations, synced across machines
---

Keep your coding agent configurations — Claude, Codex, Gemini, OpenCode — in a single canonical store. `mine agents` manages a git-backed directory that distributes shared instructions, skills, commands, and settings to every agent on your machine.

## Key Capabilities

- **Canonical store** — one source of truth at `~/.local/share/mine/agents/`
- **Git-backed** — full version history with commit, log, and restore
- **Remote sync** — push/pull to a git remote for multi-machine use
- **Structured layout** — dedicated directories for instructions, skills, commands, settings, MCP configs, and rules
- **XDG-compliant** — follows standard Linux directory conventions

## Quick Example

```bash
# Initialize the canonical agent config store
mine agents init

# Set up a remote for multi-machine sync
mine agents sync remote git@github.com:you/agent-configs.git

# Push your configs to the remote
mine agents sync push

# On another machine: pull and distribute
mine agents sync pull
```

## How It Works

`mine agents init` creates a canonical store with a scaffolded directory structure:

```
~/.local/share/mine/agents/
├── .git/
├── .mine-agents         # Manifest: links and detected agents
├── instructions/
│   └── AGENTS.md        # Shared instructions for all agents
├── skills/
├── commands/
├── agents/
├── settings/
├── mcp/
└── rules/
```

The manifest tracks which agent directories each file is linked to, and whether the link is a symlink or a file copy.

## Multi-Machine Workflow

```bash
# Machine A: set up and push
mine agents init
mine agents sync remote git@github.com:you/agent-configs.git
mine agents sync push

# Machine B: clone and link
mine agents init
mine agents sync remote git@github.com:you/agent-configs.git
mine agents sync pull
```

After pulling, copy-mode links are automatically re-distributed to their target agent directories. Symlink-mode links are already up-to-date since they point directly into the canonical store.
