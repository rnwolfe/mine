# mine — Your Personal Developer Supercharger

> "Everything you need. Nothing you don't. Radically yours."

## What is mine?

`mine` is a single CLI tool that replaces the sprawl of developer productivity tools with one fast, delightful, opinionated companion. It turbocharges development velocity, tames environment chaos, and brings a little joy to the terminal.

It's yours. It's personal. It's **mine**.

## Design Principles

1. **Speed is a feature.** Every command responds in <50ms. No spinners for local ops. Ever.
2. **One binary, zero deps.** `curl | sh` and you're done. No runtimes, no package managers.
3. **Opinionated but escapable.** Smart defaults that work for 90% of cases. Escape hatches for the rest.
4. **Whimsical but competent.** Fun personality in messages. Dead serious about getting things done.
5. **Local first.** Your data stays on your machine. Cloud features are opt-in.
6. **Progressive disclosure.** Simple surface, deep capabilities. `mine` shows you exactly what you need.
7. **Composable.** Every command works standalone and plays well with pipes, scripts, and automation.

## Brand Identity

The name `mine` is possessive — this tool is *yours*. Your config, your workflow,
your way. It's personal, opinionated, and unapologetically for you.

Command names (`dig`, `craft`, `vault`, `stash`) use common developer vocabulary
that happens to be evocative. They don't require a metaphor to understand.

## Command Map

| Command | Description | Phase | Status |
|---------|-------------|-------|--------|
| `mine` | Dashboard — your world at a glance | 1 | Shipped |
| `mine init` | First-time setup, guided onboarding | 1 | Shipped |
| `mine todo` | Fast task management with priorities | 1 | Shipped |
| `mine stash` | Dotfile & environment management | 1 | Shipped |
| `mine config` | Configuration management (get/set/list) | 1 | Shipped |
| `mine version` | Build-time version info | 1 | Shipped |
| `mine craft` | Project scaffolding & dev tool bootstrap | 2 | Shipped |
| `mine dig` | Deep work / focus mode tools | 2 | Shipped |
| `mine shell` | Shell aliases, functions, completions | 2 | Shipped |
| `mine ai` | AI tool integrations (Claude, OpenAI, etc.) | 2 | Shipped |
| `mine env` | Encrypted per-project environment profiles | 2 | Shipped |
| `mine proj` | Project registry & context switching | 2 | Shipped |
| `mine git` | Git helpers (sweep, PR, changelog, wip) | 2 | Shipped |
| `mine tmux` | Tmux session & layout management | 2 | Shipped |
| `mine ssh` | SSH connection management | 2 | Shipped |
| `mine hook` | Hook pipeline management | 2 | Shipped |
| `mine tips` | Contextual tips | 2 | Shipped |
| `mine doctor` | Health check & diagnostics | 2 | Shipped |
| `mine meta` | Interact with mine-as-a-product (feature requests, bug reports) | 2 | Shipped |
| `mine about` | About / build information | 2 | Shipped |
| `mine status` | Mine status for shell prompt integration | 2 | Shipped |
| `mine contrib` | Community contribution helpers | 2 | Shipped |
| `mine plugin` | Plugin system (install, search, manage) | 3 | Shipped |
| `mine vault` | Secrets & credential management | 3 | Shipped |
| `mine agents` | Coding agent configuration manager | 2 | Planned |
| `mine grow` | Career growth tracking & learning | 3 | Planned |
| `mine dash` | Full TUI dashboard | 3 | Planned |

## Tech Stack

- **Language**: Go 1.25+
- **CLI Framework**: Cobra
- **TUI**: Bubbletea + Lipgloss + Bubbles + Huh
- **Storage**: SQLite (pure Go, no CGo via modernc.org/sqlite)
- **Config**: TOML (human-friendly, developer-native)
- **Distribution**: Single static binary, homebrew, apt, AUR, nix

## Architecture

```
mine (binary)
├── cmd/              # Cobra command definitions (thin layer)
├── internal/
│   ├── ai/           # Multi-provider AI integrations
│   ├── analytics/    # Anonymous usage telemetry
│   ├── config/       # TOML config management (~/.config/mine/)
│   ├── contrib/      # Community contribution helpers
│   ├── craft/        # Recipe-driven scaffolding engine
│   ├── env/          # Encrypted per-project env profiles
│   ├── git/          # Git helpers (sweep, PR, changelog)
│   ├── hook/         # Four-stage hook pipeline
│   ├── meta/         # Feature request and bug report formatting
│   ├── plugin/       # Plugin system (manifest, lifecycle, runtime)
│   ├── proj/         # Project registry + context switching
│   ├── shell/        # Completions, functions, shell init
│   ├── ssh/          # SSH connection management
│   ├── stash/        # Dotfile tracking + git-backed history
│   ├── store/        # SQLite data layer (~/.local/share/mine/)
│   ├── tips/         # Contextual tips system
│   ├── tmux/         # Session + layout management
│   ├── todo/         # Todo domain logic
│   ├── tui/          # Reusable fuzzy-search picker
│   ├── ui/           # Theme, styles, print helpers
│   ├── vault/        # Age-encrypted secrets store
│   └── version/      # Build metadata
├── site/             # Astro Starlight documentation site
├── docs/             # Internal specs, plans, decisions
└── scripts/          # Build, install, release, autodev helpers
```

## Data Locations (XDG-compliant)

- Config: `~/.config/mine/config.toml`
- Data: `~/.local/share/mine/mine.db`
- Cache: `~/.cache/mine/`
- State: `~/.local/state/mine/`

## Personality

mine speaks to you like a competent friend who happens to be a wizard:

```
$ mine
▸ Hey! Here's your world:

  📋 3 todos (1 overdue — yikes)
  🔧 Node 22 + Go 1.25 + Python 3.14 ready
  📦 2 projects active
  🔑 Vault locked (3 secrets stored)
  🌱 5-day streak on learning goals

  Tip: `mine todo` to knock out that overdue task.
```

Not:
```
Mine CLI v0.1.0
Status: OK
Tasks: 3
Environment: configured
```
