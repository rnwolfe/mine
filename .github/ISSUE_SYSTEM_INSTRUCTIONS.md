# Add system instructions support for AI providers

## Overview

Allow users to configure custom system instructions (system prompts) for AI interactions. This would enable users to customize the AI's behavior, style, and domain expertise.

## Proposed Solution

Add system instructions configuration that can be:
1. Set globally in `config.toml` as `ai.system_instructions`
2. Overridden per command via `--system` flag
3. Applied to all AI requests automatically

## Configuration Example

```toml
[ai]
provider = "claude"
model = "claude-sonnet-4-5-20250929"
system_instructions = """
You are an expert Go developer with deep knowledge of clean architecture and testing.
Keep responses concise and code-focused. Always suggest tests for new code.
"""
```

## CLI Usage

```bash
# Use default system instructions from config
mine ai ask "how do I implement a retry pattern?"

# Override with command-specific instructions
mine ai review --system "Focus only on security vulnerabilities"

# Disable system instructions
mine ai ask "hello" --system ""
```

## Implementation Notes

- Store in `internal/config/config.go` as `AI.SystemInstructions string`
- Add `--system` flag to all ai subcommands
- Merge flag value with config default (flag takes precedence)
- Pass to `Request.System` field before provider calls

## Acceptance Criteria

- [ ] System instructions can be set in config.toml
- [ ] `--system` flag overrides config value
- [ ] Applied to all AI provider requests
- [ ] Empty string disables system instructions
- [ ] Documented in `mine ai --help`

## Priority

Medium - Nice to have for customization but not critical

## Labels

enhancement, phase/2
