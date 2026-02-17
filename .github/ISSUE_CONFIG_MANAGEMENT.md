# Add config get/set/list commands for easy configuration management

## Overview

Currently, users must manually edit `config.toml` or use command-specific flags to change configuration. Provide a unified `mine config` command with subcommands for viewing and modifying config values.

## Proposed Commands

```bash
# List all configuration values
mine config list

# Get a specific config value
mine config get ai.provider
mine config get user.name

# Set a config value
mine config set ai.provider claude
mine config set ai.model claude-sonnet-4-5-20250929

# Reset a value to default
mine config unset ai.model

# Open config file in $EDITOR
mine config edit
```

## Implementation Notes

- Use dot notation for nested keys (e.g., `ai.provider`, `shell.default_shell`)
- Validate values before saving (e.g., provider must be registered)
- Pretty-print output with ui styles
- Support both string values and structured output (JSON flag?)

## Example Output

```
$ mine config list

User:
  name: Ryan Wolfe

AI:
  provider: claude
  model: claude-sonnet-4-5-20250929

Shell:
  default_shell: /bin/bash

Config file: /home/user/.config/mine/config.toml
```

## Acceptance Criteria

- [ ] `mine config list` shows all config values
- [ ] `mine config get <key>` retrieves a specific value
- [ ] `mine config set <key> <value>` updates a value and saves
- [ ] `mine config unset <key>` removes a value
- [ ] `mine config edit` opens config in $EDITOR
- [ ] Validates values before saving
- [ ] Pretty output with ui styles
- [ ] Tab completion for config keys

## Priority

Medium - Quality of life improvement, config.toml editing works but less user-friendly

## Labels

enhancement, phase/2
