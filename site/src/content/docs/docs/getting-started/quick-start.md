---
title: Quick Start
description: Get up and running with mine in minutes
---

This guide walks you through the essential features of mine to get you productive quickly.

## Initialize

First, run the setup wizard:

```bash
mine init
```

This creates your config and database, auto-detecting your name from git.

## Your Dashboard

Just run `mine` with no arguments to see your personal dashboard:

```bash
mine
```

Output:
```
⛏ Hey Ryan!

  📋 Todos      3 open (1 overdue!)
  📅 Today      Monday, January 15
  ⚙️  Mine      0.1.0

  tip: `mine todo` to tackle that overdue task.
```

## Manage Tasks

Add your first todo:

```bash
mine todo add "ship v0.2" -p high -d tomorrow
```

List all todos:

```bash
mine todo
```

Mark one as done:

```bash
mine todo done 1
```

## Focus Sessions

Start a 25-minute deep work session:

```bash
mine dig
```

Or customize the duration:

```bash
mine dig 45m
```

Press `Ctrl+C` to end early (sessions over 5 minutes still count toward your streak).

## Scaffold a Project

Bootstrap a new Go project:

```bash
mkdir myproject && cd myproject
mine craft dev go
```

This creates:
- `go.mod` with the module name set to your directory
- `main.go` with a basic "Hello, world!" program
- `Makefile` with common build targets

See all available scaffolding templates:

```bash
mine craft list
```

## Track Dotfiles

Initialize dotfile tracking:

```bash
mine stash init
```

Track important config files:

```bash
mine stash track ~/.zshrc
mine stash track ~/.gitconfig
```

Check for changes:

```bash
mine stash diff
```

## Shell Integration

Generate completions for your shell:

```bash
mine shell completions
```

View recommended aliases:

```bash
mine shell aliases
```

Add them to your shell config:

```bash
alias m='mine'
alias mt='mine todo'
alias md='mine dig'
```

## Next Steps

- Explore the [command reference](/docs/commands/todo/) for all available commands
- Check out the [architecture docs](/docs/contributors/architecture/) if you want to contribute
- Join the [GitHub discussions](https://github.com/rnwolfe/mine/discussions) to share feedback
