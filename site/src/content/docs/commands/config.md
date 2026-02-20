---
title: mine config
description: View and manage configuration
---

View and manage your `mine` configuration via CLI — no manual TOML editing required.

## Show Config

```bash
mine config
```

Displays a summary of the current configuration: name, shell, AI provider, and analytics status.

## List All Keys

```bash
mine config list
```

Lists every known config key with its current value and type (`string`, `int`, `bool`).

```
user.name                        (unset)  [string]
user.email                       (unset)  [string]
ai.model                         claude-sonnet-4-5-20250929  [string]
ai.provider                      claude   [string]
ai.system_instructions           (unset)  [string]
analytics                        true     [bool]
...
```

## Get a Value

```bash
mine config get <key>
```

Returns the current value for a known key. Exits with a non-zero code and lists valid keys if the key is unknown.

```bash
mine config get ai.provider   # → claude
mine config get analytics     # → true
mine config get user.name     # → (empty if unset)
```

## Set a Value

```bash
mine config set <key> <value>
```

Sets a known configuration key with type-aware validation. Exits with a non-zero code on unknown key or type mismatch.

### Supported Keys

| Key | Type | Description |
|-----|------|-------------|
| `user.name` | string | Your display name |
| `user.email` | string | Your email address |
| `shell.default_shell` | string | Default shell path (e.g. `/bin/bash`) |
| `ai.provider` | string | AI provider (`claude`, `openai`, `gemini`, `openrouter`) |
| `ai.model` | string | AI model name |
| `ai.system_instructions` | string | Default system instructions for all AI commands |
| `ai.ask_system_instructions` | string | System instructions for `mine ai ask` |
| `ai.review_system_instructions` | string | System instructions for `mine ai review` |
| `ai.commit_system_instructions` | string | System instructions for `mine ai commit` |
| `analytics` | bool | Enable anonymous usage analytics |

### Examples

```bash
# Set your display name
mine config set user.name "Jane"

# Switch AI provider
mine config set ai.provider openai

# Set a custom AI model
mine config set ai.model gpt-4o

# Disable analytics
mine config set analytics false

# Set global AI system instructions
mine config set ai.system_instructions "Always respond in English."

# Set per-command AI system instructions
mine config set ai.ask_system_instructions "You are a Go expert."
mine config set ai.review_system_instructions "Focus on security and performance."
mine config set ai.commit_system_instructions "Use Angular commit convention."
```

### Type Validation

- **bool**: accepts `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off`
- **string**: accepts any value
- Type mismatch returns a non-zero exit code with expected-type guidance

## Unset a Value

```bash
mine config unset <key>
```

Resets a known key to its schema default. Exits with a non-zero code on unknown key.

```bash
mine config unset ai.provider   # resets to 'claude'
mine config unset user.name     # resets to empty
mine config unset analytics     # resets to true
```

## Edit Config File

```bash
mine config edit
```

Opens the config file in `$EDITOR`. If `$EDITOR` is not set, prints the config file path with instructions.

```bash
# Set your editor first
export EDITOR=vim

# Then open the config
mine config edit
```

## Show Config File Path

```bash
mine config path
```

Prints the absolute path to your config file (typically `~/.config/mine/config.toml`).

```bash
# Useful for scripting
cat $(mine config path)
```

## Hook Observability

All config commands are hook-wrapped and plugin-observable:

| Command path | Description |
|---|---|
| `config` | Bare `mine config` (dashboard) |
| `config.list` | List all keys |
| `config.get` | Get a key |
| `config.set` | Set a key |
| `config.unset` | Unset a key |
| `config.edit` | Open editor |
| `config.path` | Print path |

## Examples

```bash
# View current config
mine config

# List all keys with values and types
mine config list

# Check a specific value
mine config get ai.provider

# Change your display name
mine config set user.name "Jane"

# Switch AI provider to OpenAI
mine config set ai.provider openai
mine config set ai.model gpt-4o

# Disable analytics
mine config set analytics false

# Reset AI provider to default (claude)
mine config unset ai.provider

# Open config in your editor
mine config edit

# Get config file path
mine config path
```
