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
mine agents adopt     # Import existing configs and replace with symlinks
mine agents link      # Symlink store configs to each agent's config dir
mine agents status    # Full health report: store, agents, link states
mine agents diff      # Show content differences from canonical store
mine agents unlink    # Restore independent configs (replace symlinks with copies)
```

## How It Works

1. **One store, many agents**: Your configurations live in
   `~/.local/share/mine/agents/` — a git-backed directory with a consistent
   layout.

2. **Detection**: `mine agents detect` checks both binary presence (PATH) and
   config directory existence. Either signal counts as "detected". Results are
   persisted to the `.mine-agents` manifest.

3. **Adopt**: `mine agents adopt` is the zero-friction migration path. It scans
   existing agent configs, copies content into the canonical store, and replaces
   originals with symlinks — so you can adopt without starting from scratch.

4. **Link engine**: `mine agents link` creates symlinks from the canonical store
   to each detected agent's config directory. Change the store once, and every
   linked agent sees the update immediately. Use `--copy` for environments that
   don't support symlinks well.

5. **Safety first**: The link engine refuses to overwrite existing regular files
   without explicit `--force`. Symlinks already pointing to the canonical store
   are updated silently. Symlinks pointing elsewhere require `--force`.

6. **Conflict detection**: When multiple agents have different content for the
   same canonical file (e.g. instruction files), adopt reports the conflict and
   skips the duplicate — leaving the decision to the user.

7. **Status and diff**: `mine agents status` re-runs detection and evaluates
   every manifest link entry — reporting linked, broken, replaced, unlinked, or
   diverged state. `mine agents diff` shows content differences for copy-mode
   and replaced links, using `git diff --no-index` for clean unified diff output.

8. **Extensible registry**: Adding a new agent in a future release requires only
   appending a new entry to the agent registry — no changes to detection logic.

## Supported Agents

| Agent | Binary | Config Dir | Instruction File |
|-------|--------|-----------|-----------------|
| Claude Code | `claude` | `~/.claude/` | `CLAUDE.md` |
| Codex | `codex` | `~/.codex/` | `AGENTS.md` |
| Gemini CLI | `gemini` | `~/.gemini/` | `GEMINI.md` |
| OpenCode | `opencode` | `~/.config/opencode/` | `AGENTS.md` |

## Commands

- [`mine agents init`](/commands/agents/) — Create the canonical store
- [`mine agents detect`](/commands/agents/) — Scan and register installed agents
- [`mine agents adopt`](/commands/agents/) — Import existing agent configs into the store
- [`mine agents link`](/commands/agents/) — Symlink store configs to agent directories
- [`mine agents unlink`](/commands/agents/) — Restore independent configs
- [`mine agents status`](/commands/agents/) — Full health report: store, agents, link states
- [`mine agents diff`](/commands/agents/) — Content differences between store and linked targets
- [`mine agents`](/commands/agents/) — Alias for `mine agents status`
