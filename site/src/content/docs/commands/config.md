---
title: mine config
description: View and manage configuration
---

View and manage configuration.

## Show Config

```bash
mine config
```

Displays the current configuration in TOML format.

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

## Examples

```bash
# View current config
mine config

# Get the config file path
mine config path

# Edit config in vim
vim $(mine config path)

# Edit config in VS Code
code $(mine config path)
```
