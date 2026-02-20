---
title: mine config
description: View and manage configuration
---

View and manage configuration.

## Show Config

```bash
mine config
```

Displays the current configuration, including your name, shell, AI provider, and analytics status.

## Set a Value

```bash
mine config set <key> <value>
```

Set a configuration value. Supported keys:

| Key | Values | Description |
|-----|--------|-------------|
| `analytics` | `true` / `false` | Enable or disable anonymous usage analytics |
| `user.name` | any string | Your display name |
| `ai.provider` | `claude`, `openai`, `gemini`, `openrouter` | AI provider |
| `ai.model` | model name | AI model to use |

## Get a Value

```bash
mine config get <key>
```

Read a single configuration value. Uses the same keys as `set`.

## Show Config File Path

```bash
mine config path
```

Prints the absolute path to your config file (typically `~/.config/mine/config.toml`).

## Edit Directly

```bash
$EDITOR $(mine config path)
```

Opens your config file in your default editor.

Some settings are only available by editing the TOML file directly. For example, AI system instructions:

```toml
[ai]
system_instructions        = "Always respond in English."
ask_system_instructions    = "You are a Go expert."
review_system_instructions = "Focus on security and performance."
commit_system_instructions = "Use Angular commit convention."
```

See [`mine ai`](/commands/ai/#system-instructions) for full details on AI system instruction precedence.

## Examples

```bash
# View current config
mine config

# Disable analytics
mine config set analytics false

# Check if analytics are enabled
mine config get analytics

# Change your display name
mine config set user.name "Jane"

# Switch AI provider
mine config set ai.provider openai

# Get the config file path
mine config path

# Edit config in vim
vim $(mine config path)
```
