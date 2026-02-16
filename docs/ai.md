# AI Commands

The `mine ai` command provides AI-powered helpers directly from the CLI. It integrates with popular AI providers to assist with common development tasks.

## Quick Start

1. Configure your AI provider:
   ```bash
   mine ai config
   ```

2. Ask a question:
   ```bash
   mine ai ask "How do I write a binary search in Go?"
   ```

3. Review staged changes:
   ```bash
   git add .
   mine ai review
   ```

4. Generate commit messages:
   ```bash
   git add .
   mine ai commit
   ```

## Configuration

### Setting up Claude (Anthropic)

The default provider is Claude. To configure:

1. Get an API key from [Anthropic Console](https://console.anthropic.com/)

2. Set it via environment variable:
   ```bash
   export ANTHROPIC_API_KEY=sk-ant-...
   ```

   Or configure interactively:
   ```bash
   mine ai config
   ```

The API key is stored securely in `~/.config/mine/keys.json` with `0600` permissions (owner read/write only).

### Changing the Model

The default model is `claude-sonnet-4-5-20250929`. To change it:

1. Edit `~/.config/mine/config.toml`:
   ```toml
   [ai]
   provider = "claude"
   model = "claude-opus-4-6"  # or any other Claude model
   ```

2. Or override per-command:
   ```bash
   mine ai ask -m claude-opus-4-6 "your question"
   ```

## Commands

### `mine ai config`

Configure AI provider settings and API keys.

```bash
mine ai config
```

This will show your current configuration and allow you to set an API key if not already configured.

### `mine ai ask`

Ask a quick question to your AI.

```bash
mine ai ask "How do I reverse a string in Python?"
```

**Flags:**
- `-s, --stream` — Stream the response (default: true)
- `-m, --model` — Override the configured model

**Examples:**
```bash
# Ask with streaming output
mine ai ask "Explain goroutines"

# Disable streaming
mine ai ask --no-stream "What is the CAP theorem?"

# Use a different model
mine ai ask -m claude-opus-4-6 "Explain quantum computing"
```

### `mine ai review`

AI-powered code review of staged changes.

```bash
git add .
mine ai review
```

The AI will analyze your staged changes and provide:
- Summary of the changes
- Potential bugs or issues
- Suggestions for improvement
- Security concerns (if any)

**Flags:**
- `-m, --model` — Override the configured model

**Example:**
```bash
git add internal/ai/claude.go
mine ai review
```

### `mine ai commit`

Generate a conventional commit message from your staged changes.

```bash
git add .
mine ai commit
```

The AI will:
1. Analyze your staged diff
2. Generate a conventional commit message (e.g., `feat: add AI provider integration`)
3. Ask if you want to use it
4. Commit with that message if you confirm

**Flags:**
- `-m, --model` — Override the configured model

**Example:**
```bash
git add cmd/ai.go internal/ai/
mine ai commit
# AI generates: "feat: add AI command with ask, review, and commit subcommands"
# Use this message? (y/n): y
# ✓ Changes committed successfully
```

## Offline Usage

All AI commands gracefully degrade when no API key is configured:

```bash
$ mine ai ask "hello"
claude: API key not found (set ANTHROPIC_API_KEY or run `mine ai config`)
```

No crashes, just clear error messages telling you what to do.

## Security

- API keys are stored in `~/.config/mine/keys.json` with `0600` permissions
- Environment variables take precedence over stored keys
- Keys are never logged or printed to the console
- API requests use HTTPS with a 2-minute timeout

## Future Providers

The provider abstraction is designed to support multiple AI providers:
- Claude (Anthropic) ✅ **Implemented**
- OpenAI (planned)
- Ollama (planned — fully offline)
- Custom local models (planned)

To request a new provider, [open an issue](https://github.com/rnwolfe/mine/issues/new).

## Troubleshooting

### "API key not found"

Set your API key:
```bash
export ANTHROPIC_API_KEY=sk-ant-...
# or
mine ai config
```

### "Request timeout"

The default timeout is 2 minutes. If you're on a slow connection or asking complex questions, the request might time out. This is a safety measure to prevent hanging indefinitely.

### "Invalid model"

Check that the model name is correct for your provider. For Claude, valid models include:
- `claude-sonnet-4-5-20250929` (default)
- `claude-opus-4-6`
- Other Claude models from the [Anthropic API docs](https://docs.anthropic.com/claude/docs/models-overview)

## Examples

### Quick code review before committing
```bash
git add .
mine ai review
# Read the feedback
mine ai commit
# Commit with AI-generated message
```

### Learning a new concept
```bash
mine ai ask "Explain the Actor model in concurrent programming"
```

### Getting unstuck
```bash
mine ai ask "I'm getting a 'nil pointer dereference' in Go. What are common causes?"
```

### Improving code quality
```bash
# Stage your changes
git add internal/mypackage/

# Get AI review
mine ai review

# Make improvements based on feedback

# Generate commit message
mine ai commit
```
