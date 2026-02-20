---
title: Configuration
description: Manage mine settings from the terminal with type-aware validation
---

Manage all `mine` settings directly from the terminal — no manual TOML editing required. The `mine config` command suite provides a typed key registry with get/set/unset operations, so you always know what values are valid and what the defaults are.

## Key Capabilities

- **List all keys** — see every configurable setting with its current value and type
- **Get/set/unset** — read and write individual keys with type validation
- **Schema defaults** — `unset` restores the documented default, not just blanks the value
- **$EDITOR integration** — open the raw TOML file when you need direct access
- **Hook-wrapped** — all config commands are observable by plugins

## Quick Example

```bash
# See every setting with current values
mine config list

# Personalize mine
mine config set user.name "Jane"
mine config set shell.default_shell /bin/zsh

# Switch AI provider and model
mine config set ai.provider openai
mine config set ai.model gpt-4o

# Set global AI system instructions
mine config set ai.system_instructions "Always respond in English."

# Opt out of analytics
mine config set analytics false

# Reset a key to its schema default
mine config unset ai.provider   # resets to 'claude'

# Open raw config in your editor
mine config edit

# Get the config file path (useful in scripts)
mine config path
```

## How It Works

All known config keys live in a typed registry inside `internal/config/`. Each key carries its type (`string`, `bool`, or `int`), a description, and a default value. When you run `mine config set`, the value is validated against the declared type before being written to disk. When you run `mine config unset`, the key is reset to its schema default — not just deleted.

The config file is standard TOML at `~/.config/mine/config.toml` (XDG-compliant). You can always edit it directly with `mine config edit` or `$EDITOR $(mine config path)`.

## Supported Keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `user.name` | string | (empty) | Your display name |
| `user.email` | string | (empty) | Your email address |
| `shell.default_shell` | string | `$SHELL` | Default shell path |
| `ai.provider` | string | `claude` | AI provider |
| `ai.model` | string | `claude-sonnet-4-5-20250929` | AI model name |
| `ai.system_instructions` | string | (empty) | Global AI system instructions |
| `ai.ask_system_instructions` | string | (empty) | System instructions for `mine ai ask` |
| `ai.review_system_instructions` | string | (empty) | System instructions for `mine ai review` |
| `ai.commit_system_instructions` | string | (empty) | System instructions for `mine ai commit` |
| `analytics` | bool | `true` | Anonymous usage analytics |

## Bool Values

The `bool` type accepts multiple formats: `true`/`false`, `1`/`0`, `yes`/`no`, `on`/`off`.

```bash
mine config set analytics false
mine config set analytics 0
mine config set analytics no    # all equivalent
```

## Learn More

See the [command reference](/commands/config/) for all subcommands, error codes, and hook paths.
