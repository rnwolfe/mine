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

## The Mining Metaphor

The name `mine` works on multiple levels:
- **Possessive**: This tool is *yours*. Your config, your workflow, your way.
- **Mining**: Digging for gold — extracting maximum value from your dev environment.
- **Crafting**: Building things — scaffolding projects, shaping environments.

Commands lean into this naturally without being forced.

## Command Map

| Command | Description | Status |
|---------|-------------|--------|
| `mine` | Dashboard — your world at a glance | Phase 1 |
| `mine init` | First-time setup, guided onboarding | Phase 1 |
| `mine todo` | Fast task management with priorities | Phase 1 |
| `mine stash` | Dotfile & environment management | Phase 1 |
| `mine craft` | Project scaffolding & dev tool bootstrap | Phase 2 |
| `mine dig` | Deep work / focus mode tools | Phase 2 |
| `mine shell` | Shell aliases, functions, completions | Phase 2 |
| `mine ai` | AI tool integrations (Claude, Copilot, etc.) | Phase 2 |
| `mine vault` | Secrets & credential management | Phase 3 |
| `mine grow` | Career growth tracking & learning | Phase 3 |
| `mine dash` | Full TUI dashboard | Phase 3 |
| `mine config` | Configuration management | Phase 1 |

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
│   ├── config/       # TOML config management (~/.config/mine/)
│   ├── store/        # SQLite data layer (~/.local/share/mine/)
│   ├── ui/           # Shared TUI components, styles, themes
│   ├── todo/         # Todo domain logic
│   ├── env/          # Environment/dotfile management
│   ├── craft/        # Project scaffolding engine
│   ├── shell/        # Shell integration layer
│   ├── ai/           # AI provider integrations
│   ├── vault/        # Encrypted secrets store
│   ├── grow/         # Growth tracking domain
│   └── dash/         # Dashboard composition
├── docs/             # All documentation
├── scripts/          # Build, install, release scripts
├── config/           # Default/example configurations
└── examples/         # Example configs and workflows
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
⛏  Hey! Here's your world:

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
