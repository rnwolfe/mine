---
title: Other Commands
description: Additional mine commands
---

## mine init

Guided first-time setup. Creates config and data directories.

```bash
mine init
```

Auto-detects your name from `~/.gitconfig`. Creates:
- `~/.config/mine/config.toml`
- `~/.local/share/mine/mine.db`

## mine config

View and manage configuration.

### Show config

```bash
mine config
```

### Show config file path

```bash
mine config path
```

### Edit directly

```bash
$EDITOR $(mine config path)
```

## mine version

Print version, commit hash, and build date.

```bash
mine version
```

## mine (dashboard)

The default command shows your personal dashboard.

```
$ mine
⛏ Hey Ryan!

  📋 Todos      3 open (1 overdue!)
  📅 Today      Sunday, February 16
  ⚙️  Mine      0.1.0

  tip: `mine todo` to tackle that overdue task.
```

If mine hasn't been initialized, it prompts you to run `mine init`.
