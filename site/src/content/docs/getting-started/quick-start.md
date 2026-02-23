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

This creates your config and database, auto-detecting your name from git. After setup, `mine init` shows a **capability table** — a dynamic readout of which features are ready based on your environment:

```
  What you've got:

    ✓  todos          — mine todo add "ship it"
    ✓  stash          — mine stash add <url>
    ✓  env            — mine env init
    ✓  git            — mine git log
    ·  tmux           — install tmux, then mine tmux new
    ✓  AI (claude)    — mine ai ask "explain this diff"
    ✓  proj           — mine proj list
```

If you run `mine init` from inside a git repository, it also offers to register the current directory as a mine project — so your dashboard has project context immediately.

## Your Dashboard

Just run `mine` with no arguments to see your personal dashboard:

```bash
mine
```

Output:
```
▸ Hey Ryan!

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

`mine init` handles first-time shell setup for you. After the AI section it shows:

```
  Shell Integration

  Adding this line to ~/.zshrc enables p, pp, and menv:

    eval "$(mine shell init)"

  Add it now? (Y/n)
```

Press Enter to let mine write the line to your RC file. This activates:

- **`p [name]`** — jump to a project (opens fuzzy picker with no args)
- **`pp`** — return to the previous project
- **`menv`** — load your active `mine env` profile into the current shell

To add it manually instead:

```bash
printf '\n# added by mine\neval "$(mine shell init)"\n' >> ~/.zshrc
source ~/.zshrc
```

For tab completions and optional aliases:

```bash
mine shell completions   # tab completions
mine shell aliases       # view recommended aliases
```

## Configure mine

Personalize your setup without editing TOML files directly:

```bash
# See all settings and their current values
mine config list

# Set your display name
mine config set user.name "Jane"

# Switch to a different AI provider
mine config set ai.provider openai
mine config set ai.model gpt-4o

# Opt out of analytics
mine config set analytics false
```

See the [config command reference](/commands/config/) for all keys and options.

## Next Steps

- Explore the [command reference](/commands/todo/) for all available commands
- Check out the [architecture docs](/contributors/architecture/) if you want to contribute
- Join the [GitHub discussions](https://github.com/rnwolfe/mine/discussions) to share feedback
