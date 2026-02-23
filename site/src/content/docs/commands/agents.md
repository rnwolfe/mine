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

## Adopt Existing Configs

```bash
mine agents adopt
```

Scans detected agents for existing configurations, imports them into the
canonical store, and replaces the originals with symlinks. This is the
zero-friction migration path for developers who already have agent configs in
place and want to start managing them centrally.

**What gets adopted:**

| Content Type | Example Source | Store Location |
|-------------|---------------|----------------|
| Instruction file | `~/.claude/CLAUDE.md` | `instructions/AGENTS.md` |
| Skills directory | `~/.claude/skills/` | `skills/` |
| Commands directory | `~/.claude/commands/` | `commands/` |
| Settings file | `~/.claude/settings.json` | `settings/claude.json` |
| MCP config | `~/.claude/.mcp.json` | `mcp/.mcp.json` |
| Agent definitions | `~/.claude/agents/` | `agents/` |
| Rules directory | `~/.claude/rules/` | `rules/` |

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent <name>` | Adopt only from a specific agent (e.g. `claude`, `codex`) |
| `--dry-run` | Show what would be imported without making any changes |
| `--copy` | Import files into the store but don't replace originals with symlinks |

**Conflict resolution:**
- First agent's instruction file sets the canonical `instructions/AGENTS.md`
- Subsequent agents with different instruction content: reported as conflict, skipped
- Subsequent agents with identical instruction content: reported as already-managed
- Settings files are always stored per-agent (`settings/<name>.json`) — no conflict possible
- Directory content is merged non-destructively: existing store files are never overwritten
- Files already managed by a symlink to the store are skipped automatically

**After adopt:**
- All imported content is committed to the store's git history with message `adopt: imported configs from <agents>`
- Originals are replaced with symlinks (unless `--copy`)
- Run `mine agents status` to verify the result

## Link Configs

```bash
mine agents link
```

Creates symlinks from the canonical store to each detected agent's expected
configuration locations. Only config types that exist in the store are linked —
empty directories are skipped automatically.

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent <name>` | Link only a specific agent (e.g. `claude`, `codex`) |
| `--copy` | Create file copies instead of symlinks |
| `--force` | Overwrite existing non-symlink files without requiring adopt first |

**Link map:**

| Config Type | Source (store-relative) | Claude Target |
|-------------|------------------------|---------------|
| Instructions | `instructions/AGENTS.md` | `~/.claude/CLAUDE.md` |
| Skills | `skills/` | `~/.claude/skills/` |
| Commands | `commands/` | `~/.claude/commands/` |
| Settings | `settings/claude.json` | `~/.claude/settings.json` |
| MCP config | `mcp/.mcp.json` | `~/.claude/.mcp.json` |

**Safety rules:**
- Existing regular files → refused; suggests `adopt` or `--force`
- Existing symlink to canonical store → updated silently
- Existing symlink pointing elsewhere → refused without `--force`
- Missing parent directory → created automatically

## Unlink Configs

```bash
mine agents unlink
```

Replaces agent config symlinks with standalone file copies, restoring each
agent's configuration to an independent state. After unlinking, changes to the
canonical store no longer propagate automatically.

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent <name>` | Unlink only a specific agent (e.g. `claude`) |

**Unlink behavior:**
- File symlinks → content read, symlink removed, standalone file written
- Directory symlinks → directory copied, symlink removed
- Copy-mode entries → only manifest tracking removed (files already standalone)

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
| `git init: exec: "git": executable file not found...` | git not in PATH | Install git |
| `reading manifest: parsing manifest` | Corrupt `.mine-agents` file | Remove and re-run `mine agents init` |
| `conflict` in adopt output | Multiple agents have different instruction content | Edit `instructions/AGENTS.md` manually, then re-run adopt |
