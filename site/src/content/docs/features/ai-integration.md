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
- **Styled markdown output** — responses rendered as formatted markdown in interactive terminals (headings, code blocks, lists, emphasis)
- **Secure key storage** — API keys stored in the encrypted vault, or use environment variables

## Quick Example

```bash
# Configure a provider (key stored in vault)
mine ai config --provider claude --key sk-ant-...

# Ask a question — response rendered as styled markdown in TTY
mine ai ask "What's the difference between a mutex and a semaphore?"

# Review staged changes — review rendered as styled markdown in TTY
git add . && mine ai review

# Generate a commit message
git add . && mine ai commit

# Force plain markdown output (useful for piping to files or other tools)
mine ai ask "Explain goroutines" --raw > answer.md
mine ai review --raw | less
```

## How It Works

Configure a provider once with `mine ai config` and the API key is stored encrypted in the vault. From then on, AI commands just work. `mine ai review` sends your staged diff to the configured provider and returns feedback. `mine ai commit` analyzes staged changes and suggests a conventional commit message, optionally running `git commit` for you.

For zero-config setups, just set the standard environment variable (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.) and mine detects it automatically. Environment variables take precedence over vault-stored keys.

Use `mine ai models` to see all available providers, their suggested models, and whether they're configured.

## Markdown Rendering

In interactive terminals (TTY), `mine ai ask` and `mine ai review` automatically render AI responses as styled markdown — headings, code blocks, lists, bold/italic text are all formatted for readability using [glamour](https://github.com/charmbracelet/glamour).

When piping output or running in a non-TTY context, raw markdown is emitted automatically — no flags needed. Use `--raw` to force plain output even in a TTY (useful when saving to a file or chaining with other tools):

```bash
# TTY: styled output (default)
mine ai ask "How does Go garbage collection work?"

# Non-TTY: raw markdown automatically
mine ai ask "How does Go garbage collection work?" | cat

# Force raw regardless of context
mine ai ask "How does Go garbage collection work?" --raw > answer.md
```

## Learn More

See the [command reference](/commands/ai/) for provider setup, model selection, and all subcommands.
