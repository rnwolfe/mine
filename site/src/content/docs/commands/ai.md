---
title: mine ai
description: AI-powered development helpers
---

Use AI to assist with code review, commit messages, and quick questions.
Supports Claude, OpenAI, Gemini, and OpenRouter.

## Configure a Provider

```bash
mine ai config --provider claude --key sk-ant-...
mine ai config --provider openai --key sk-...
mine ai config --provider gemini --key AIza...
mine ai config --provider openrouter --key sk-or-v1-...
```

API keys are stored encrypted in the vault (`~/.local/share/mine/vault.age`).
See [`mine vault`](/commands/vault) for details.

### Set Default Model

```bash
mine ai config --provider claude --default-model claude-opus-4-6
```

### List Configured Providers

```bash
mine ai config --list
```

## Zero-Config Setup

Set a standard environment variable and mine detects it automatically:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
export GEMINI_API_KEY=AIza...
export OPENROUTER_API_KEY=sk-or-v1-...
```

Environment variables take precedence over vault-stored keys.

## Ask a Question

```bash
mine ai ask "What is the difference between defer and panic in Go?"
mine ai ask "Explain the repository pattern" --model claude-opus-4-6
mine ai ask "What does this code do?" --system "You are a Go expert. Be concise."
```

In interactive terminals (TTY), responses are automatically rendered as styled markdown — headings, code blocks, lists, and emphasis are all formatted for readability.

### Raw output

Use `--raw` to force plain markdown output (no terminal rendering). Useful for piping into other tools or saving to a file:

```bash
mine ai ask "Explain goroutines" --raw
mine ai ask "Explain goroutines" --raw > answer.md
mine ai ask "Explain goroutines" | cat   # non-TTY also outputs raw automatically
```

## Review Staged Changes

```bash
git add .
mine ai review
mine ai review --system "Focus only on security issues."
mine ai review --system ""   # disable system instructions for this review
```

Sends your staged git diff to the configured AI provider for code review. Output is rendered as styled markdown in interactive terminals.

### Raw output

```bash
mine ai review --raw             # force plain markdown
mine ai review --raw > review.md # pipe to file
```

## Generate a Commit Message

```bash
git add .
mine ai commit
mine ai commit --system "Use Angular commit convention."
```

Analyzes staged changes, suggests a conventional commit message, and optionally runs `git commit`.

## System Instructions

Control what behavior the AI uses for each command via the `--system` flag or config defaults.

### Flags

| Flag | Description |
|------|-------------|
| `--system "<text>"` | Override system instructions for this invocation |
| `--system ""` | Disable system instructions for this invocation |

### Config Keys

Set persistent defaults using `mine config set`:

```bash
# Global default applied to all AI subcommands
mine config set ai.system_instructions "Always respond in English."

# Per-subcommand defaults (override the global default)
mine config set ai.ask_system_instructions "You are a Go expert."
mine config set ai.review_system_instructions "Focus on security and performance."
mine config set ai.commit_system_instructions "Use Angular commit convention."
```

Or edit the TOML file directly (`mine config edit`):

```toml
[ai]
system_instructions        = "Always respond in English."
ask_system_instructions    = "You are a Go expert."
review_system_instructions = "Focus on security and performance."
commit_system_instructions = "Use Angular commit convention."
```

### Precedence Rules

System instructions are resolved in this order (first match wins):

1. `--system` flag (including empty string, which disables system instructions)
2. Per-subcommand config key (`ask_system_instructions`, etc.)
3. Global config key (`system_instructions`)
4. Built-in default (for `review` and `commit` only)

When no custom value is configured or provided, `review` and `commit` use their built-in system prompts. `ask` has no built-in default.

## List Providers and Models

```bash
mine ai models
```

Shows all available providers with their suggested models and configuration status.

## API Key Storage

API keys configured via `mine ai config --key` are stored in the encrypted vault:

| Provider | Vault key |
|----------|-----------|
| claude | `ai.claude.api_key` |
| openai | `ai.openai.api_key` |
| gemini | `ai.gemini.api_key` |
| openrouter | `ai.openrouter.api_key` |

You can also manage these keys directly with `mine vault`:

```bash
mine vault set ai.claude.api_key sk-ant-...
mine vault get ai.claude.api_key
mine vault rm ai.claude.api_key
```

## Supported Providers

| Provider | Env Var | Notes |
|----------|---------|-------|
| `claude` | `ANTHROPIC_API_KEY` | [Get key](https://console.anthropic.com/settings/keys) |
| `openai` | `OPENAI_API_KEY` | [Get key](https://platform.openai.com/api-keys) |
| `gemini` | `GEMINI_API_KEY` | [Get key](https://aistudio.google.com/app/apikey) |
| `openrouter` | `OPENROUTER_API_KEY` | Free models available. [Get key](https://openrouter.ai/keys) |
