# mine ⛏️

> Your personal developer supercharger.

A single CLI tool that turbocharges dev velocity, tames environment chaos, and brings a little joy to the terminal. One binary. Zero deps. Radically fast.

## Install

```bash
# Quick install
curl -fsSL https://raw.githubusercontent.com/rnwolfe/mine/main/scripts/install.sh | bash

# Or build from source
go install github.com/rnwolfe/mine@latest
```

## What it does

```
$ mine
⛏ Hey Ryan!

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

Config lives at `~/.config/mine/config.toml` (XDG-compliant):

```toml
[user]
name = "Ryan"

[shell]
default_shell = "/usr/bin/zsh"

[ai]
provider = "claude"
model = "claude-sonnet-4-5-20250929"
```

Data: `~/.local/share/mine/mine.db` (SQLite)

## Tech

- **Go 1.25+** — fast compilation, single binary output
- **Cobra** — CLI framework
- **Lipgloss** — terminal styling (Charm ecosystem)
- **SQLite** — pure Go, WAL mode, no CGo
- **TOML** — human-friendly config

## Development

```bash
make build    # build to bin/mine
make test     # run tests
make lint     # go vet
make dev ARGS="todo"  # quick dev cycle
```

## License

MIT
