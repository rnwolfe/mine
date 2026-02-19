---
title: AI Integration
description: Multi-provider AI for code review, commit messages, and quick questions
---

Get AI assistance without leaving the terminal. `mine ai` supports Claude, OpenAI, Gemini, and OpenRouter for code review, commit message generation, and general questions — with keys stored securely in the vault.

## Key Capabilities

- **Multi-provider** — Claude, OpenAI, Gemini, and OpenRouter supported out of the box
- **Code review** — send staged git diffs for AI review in one command
- **Commit messages** — generate conventional commit messages from staged changes
- **Quick questions** — ask coding questions directly from the terminal
- **Secure key storage** — API keys stored in the encrypted vault, or use environment variables
- **System instructions** — customize AI behavior globally or per-subcommand via config or `--system` flag

## Quick Example

```bash
# Configure a provider (key stored in vault)
mine ai config --provider claude --key sk-ant-...

# Ask a question
mine ai ask "What's the difference between a mutex and a semaphore?"

# Review staged changes
git add . && mine ai review

# Generate a commit message
git add . && mine ai commit

# Override AI behavior for a single invocation
mine ai review --system "Focus only on security issues."
mine ai ask "Explain Go channels" --system "You are a Go expert. Be concise."
```

## How It Works

Configure a provider once with `mine ai config` and the API key is stored encrypted in the vault. From then on, AI commands just work. `mine ai review` sends your staged diff to the configured provider and returns feedback. `mine ai commit` analyzes staged changes and suggests a conventional commit message, optionally running `git commit` for you.

For zero-config setups, just set the standard environment variable (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.) and mine detects it automatically. Environment variables take precedence over vault-stored keys.

System instructions let you tailor how the AI responds. Pass `--system "<text>"` on any `ai` subcommand to override behavior for that invocation, or set persistent defaults in `~/.config/mine/config.toml` under `[ai]`. A four-level precedence chain (flag → subcommand config → global config → built-in default) gives you fine-grained control without breaking existing workflows.

Use `mine ai models` to see all available providers, their suggested models, and whether they're configured.

## Learn More

See the [command reference](/commands/ai/) for provider setup, model selection, and all subcommands.
