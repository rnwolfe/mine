# mine

> Your personal developer supercharger.

[![CI](https://github.com/rnwolfe/mine/actions/workflows/ci.yml/badge.svg)](https://github.com/rnwolfe/mine/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/rnwolfe/mine/main/docs/internal/badge-coverage.json)](https://github.com/rnwolfe/mine/blob/main/docs/internal/coverage.json)
[![Release](https://img.shields.io/github/v/release/rnwolfe/mine)](https://github.com/rnwolfe/mine/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/github/license/rnwolfe/mine)](LICENSE)

A single CLI tool that turbocharges dev velocity, tames environment chaos, and brings a little joy to the terminal. One binary. No runtime dependencies. Some features integrate with tools you already have (git, tmux, ssh).

## Install

```bash
# Quick install
curl -fsSL https://mine.rwolfe.io/install | bash

# Or build from source
go install github.com/rnwolfe/mine@latest
```

## What it does

```
$ mine
▸ Hey Ryan!

  📋 Todos      3 open (1 overdue!)
  📅 Today      Friday, February 14
  ⚙️  Mine      0.1.0

  tip: `mine todo` to tackle that overdue task.
```

| Command | What | Example |
|---------|------|---------|
| `mine` | Dashboard | `mine` |
| `mine todo` | Task management | `mine todo add "ship it" -p high -d tomorrow` |
| `mine stash` | Dotfile tracking | `mine stash track ~/.zshrc` |
| `mine craft` | Project scaffolding | `mine craft dev go` |
| `mine dig` | Focus timer | `mine dig 25m` |
| `mine shell` | Shell integration | `mine shell completions zsh` |
| `mine config` | Configuration | `mine config` |

## Todos

```bash
mine todo add "build the thing" -p high -d tomorrow
mine todo add "fix that bug" -p crit --tags "backend,urgent"
mine todo                        # list open tasks
mine todo done 1                 # complete a task
mine todo rm 2                   # delete a task
mine todo --all                  # show completed tasks too
```

Priorities: `low` `med` `high` `crit`
Due dates: `today` `tomorrow` `next-week` `2026-03-01`

## Focus Timer

```bash
mine dig          # 25m pomodoro (default)
mine dig 45m      # custom duration
mine dig 1h       # longer session
mine dig stats    # see your streak
```

## Dotfiles

```bash
mine stash init                  # initialize dotfile tracking
mine stash track ~/.zshrc        # start tracking a file
mine stash track ~/.gitconfig    # track another
mine stash list                  # see what's tracked
mine stash diff                  # check for changes
```

## Project Scaffolding

```bash
mine craft dev go       # bootstrap a Go project
mine craft dev node     # bootstrap Node.js
mine craft dev python   # bootstrap Python
mine craft git          # set up git with .gitignore
```

## Shell Integration

```bash
mine shell completions zsh    # generate completions
mine shell completions bash
mine shell completions fish
mine shell aliases            # see recommended aliases
```

## Configuration

Manage settings via the CLI — no manual TOML editing required:

```bash
mine config list                           # see all keys and current values
mine config get ai.provider               # check a value
mine config set user.name "Jane"          # set your display name
mine config set ai.provider openai        # switch AI provider
mine config set analytics false           # opt out of analytics
mine config unset ai.provider             # reset to default (claude)
mine config edit                          # open in $EDITOR
mine config path                          # show config file location
```

Config file: `~/.config/mine/config.toml` (XDG-compliant)
Data: `~/.local/share/mine/mine.db` (SQLite)

## Tech

- **Go 1.25+** — fast compilation, single binary output
- **Cobra** — CLI framework
- **Lipgloss** — terminal styling (Charm ecosystem)
- **SQLite** — pure Go, WAL mode, no CGo
- **TOML** — human-friendly config

## Documentation

Full documentation available at **[mine.rwolfe.io](https://mine.rwolfe.io)**:

- [Installation Guide](https://mine.rwolfe.io/getting-started/installation/)
- [Quick Start](https://mine.rwolfe.io/getting-started/quick-start/)
- [Command Reference](https://mine.rwolfe.io/commands/todo/)
- [Architecture](https://mine.rwolfe.io/contributors/architecture/)
- [Plugin Protocol](https://mine.rwolfe.io/contributors/plugin-protocol/)

## Development

```bash
make build    # build to bin/mine
make test     # run tests
make lint     # go vet
make dev ARGS="todo"  # quick dev cycle
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for full development workflow.

## Autonomous Development

mine is built by an AI-powered development pipeline. When an issue is labeled `backlog/ready`, it's automatically picked up, implemented, reviewed (Copilot + Claude), and merged — no human required until it ships to users.

[![Pipeline Health](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/rnwolfe/mine/main/docs/internal/badge-pipeline-health.json)](https://github.com/rnwolfe/mine/issues?q=label%3Areport%2Fpipeline-audit)
[![Autodev PRs](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/rnwolfe/mine/main/docs/internal/badge-autodev.json)](https://github.com/rnwolfe/mine/pulls?q=label%3Avia%2Factions+is%3Amerged)

The pipeline implements issues, runs weekly quality sweeps (docs, code health, backlog, personality), and automatically proposes releases when enough features accumulate. See [docs/internal/autodev-pipeline.md](docs/internal/autodev-pipeline.md) for the full architecture.

## License

MIT
