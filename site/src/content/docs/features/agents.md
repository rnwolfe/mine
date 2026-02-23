---
title: Agents
description: One canonical store for all your coding agent configurations — instructions, skills, settings, and more — symlinked everywhere
---

You have Claude Code. Then you install Codex. Then Gemini CLI. Then OpenCode. Before long,
you have four different directories full of instruction files, skills, and settings that
all contain slight variations of the same content — and you have no idea which one is
"real" anymore.

`mine agents` solves this with a single canonical store: one place for everything, linked
everywhere it needs to be.

## The Problem

Coding agents each expect their configuration in a different place:

| Agent | Instruction File | Config Directory |
|-------|-----------------|-----------------|
| Claude Code | `~/.claude/CLAUDE.md` | `~/.claude/` |
| Codex | `~/.codex/AGENTS.md` | `~/.codex/` |
| Gemini CLI | `~/.gemini/GEMINI.md` | `~/.gemini/` |
| OpenCode | `~/.config/opencode/AGENTS.md` | `~/.config/opencode/` |

Writing the same instructions four times is bad enough. Keeping them in sync as you update
them is worse. `mine agents` eliminates this entirely.

## The Solution

A single git-backed store at `~/.local/share/mine/agents/` becomes the source of truth.
Every agent's config directory gets a symlink pointing back to it. Edit once, every agent
sees the change — immediately.

```
~/.local/share/mine/agents/       ← canonical store (yours to edit)
├── instructions/
│   └── AGENTS.md                 ← shared instructions for all agents
├── skills/                       ← shared skills
├── commands/                     ← Claude-specific slash commands
├── settings/
│   ├── claude.json
│   └── codex.json
└── mcp/
    └── .mcp.json

~/.claude/CLAUDE.md  →  symlink to ~/.local/share/mine/agents/instructions/AGENTS.md
~/.codex/AGENTS.md   →  symlink to ~/.local/share/mine/agents/instructions/AGENTS.md
~/.gemini/GEMINI.md  →  symlink to ~/.local/share/mine/agents/instructions/AGENTS.md
```

## Quick Start

Starting from scratch:

```bash
# 1. Create the canonical store
mine agents init

# 2. Detect what agents you have installed
mine agents detect

# 3. Distribute configs to every detected agent
mine agents link
```

Already have configs scattered across your agents? Adopt them:

```bash
# Import existing configs into the store and replace with symlinks
mine agents adopt

# Check what was imported and verify link health
mine agents status
```

## Typical Workflow

```bash
# One-time setup
mine agents init
mine agents detect
mine agents adopt     # import what you already have
mine agents sync remote git@github.com:you/agent-configs.git
mine agents sync push

# Daily use — edit store once, all agents see it
$EDITOR ~/.local/share/mine/agents/instructions/AGENTS.md

# Snapshot your changes
mine agents commit -m "add project scaffolding instructions"

# Sync to another machine
mine agents sync push

# On the other machine
mine agents sync pull
```

## What Gets Linked

The link engine distributes every config type that exists in your store, mapped to
each agent's expected location:

| Config Type | Canonical Location | Claude Target | Codex Target |
|-------------|-------------------|---------------|--------------|
| Instructions | `instructions/AGENTS.md` | `~/.claude/CLAUDE.md` | `~/.codex/AGENTS.md` |
| Skills | `skills/` | `~/.claude/skills/` | `~/.codex/skills/` |
| Commands | `commands/` | `~/.claude/commands/` | — |
| Settings | `settings/claude.json` | `~/.claude/settings.json` | — |
| MCP config | `mcp/.mcp.json` | `~/.claude/.mcp.json` | — |

Empty directories are skipped. Missing config types are silently ignored. Every detected
agent gets exactly what it supports.

## The AGENTS.md Standard

The instructions file in the canonical store uses the
[AGENTS.md](https://github.com/google-deepmind/agent-instructions-interchange) standard
— a cross-agent portable format for coding agent instructions. This means your shared
instructions work natively with Claude Code (as `CLAUDE.md`), Codex (as `AGENTS.md`),
Gemini CLI (as `GEMINI.md`), and any other agent that reads local instruction files.

## Project-Level Configs

For project-specific configurations, `mine agents project` scaffolds agent config dirs
inside a project and optionally links shared skills from the canonical store:

```bash
# Scaffold project-level agent dirs
mine agents project init ~/projects/myapp

# Share canonical skills with the project
mine agents project link ~/projects/myapp
```

See the [command reference](/commands/agents/) for the full project subcommand reference.

## Version History

The canonical store is a git repository from day one. Every adopt is auto-committed.
You can also snapshot manually:

```bash
mine agents commit -m "add new formatting rules"
mine agents log
mine agents restore instructions/AGENTS.md --version abc1234
```

## Multi-Machine Sync

Push the store to a git remote and pull it on any machine:

```bash
mine agents sync remote git@github.com:you/agent-configs.git
mine agents sync push   # after changes
mine agents sync pull   # on another machine (re-distributes copy-mode links automatically)
```

## Learn More

See the [command reference](/commands/agents/) for all subcommands, flags, error codes,
and the full link distribution map.
