---
title: Agent Configuration Management
description: One canonical store for all your coding agent instructions, rules, and skills
---

`mine agents` gives you a single canonical store for all your coding agent
configurations — shared instructions, rules, skills, and settings — versioned
in git and ready to sync across Claude Code, Codex, Gemini CLI, and OpenCode.

## Quick Example

```bash
mine agents init      # Create the canonical store
mine agents detect    # Scan for installed agents
mine agents           # Show status
```

## How It Works

1. **One store, many agents**: Your configurations live in
   `~/.local/share/mine/agents/` — a git-backed directory with a consistent
   layout.

2. **Detection**: `mine agents detect` checks both binary presence (PATH) and
   config directory existence. Either signal counts as "detected". Results are
   persisted to the `.mine-agents` manifest.

3. **Extensible registry**: Adding a new agent in a future release requires only
   appending a new entry to the agent registry — no changes to detection logic.

## Supported Agents

| Agent | Binary | Config Dir |
|-------|--------|-----------|
| Claude Code | `claude` | `~/.claude/` |
| Codex | `codex` | `~/.codex/` |
| Gemini CLI | `gemini` | `~/.gemini/` |
| OpenCode | `opencode` | `~/.config/opencode/` |

## Commands

- [`mine agents init`](/commands/agents/) — Create the canonical store
- [`mine agents detect`](/commands/agents/) — Scan and register installed agents
- [`mine agents`](/commands/agents/) — Show store status
