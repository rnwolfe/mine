# mine — Project Status

## v0.1.0 — Phase 1: Foundation (Current)

### Done

- [x] **Core framework** — Cobra CLI, Go module, build system
- [x] **Theme system** — Lipgloss-based colors, icons, styled output
- [x] **Config** — TOML config, XDG-compliant paths, load/save
- [x] **SQLite store** — WAL mode, migrations, high-perf pragmas
- [x] **Dashboard** (`mine`) — At-a-glance status with greeting, todos, tips
- [x] **Init** (`mine init`) — Guided onboarding, git name detection
- [x] **Todos** (`mine todo`) — Add, complete, delete, edit, list with priority/due/tags
- [x] **Stash** (`mine stash`) — Dotfile tracking, diff detection, manifest
- [x] **Craft** (`mine craft`) — Git setup, Go/Node/Python project scaffolding
- [x] **Dig** (`mine dig`) — Pomodoro focus timer with streaks and stats
- [x] **Shell** (`mine shell`) — Completions (bash/zsh/fish), alias suggestions
- [x] **Version** (`mine version`) — Build-time version injection via ldflags
- [x] **Tests** — 10 passing (todo + config domain tests)
- [x] **Install script** — curl | bash installer for releases

### Binary Stats

- **Size**: 7.6 MB (stripped)
- **Deps**: 0 runtime (single static binary)
- **Build time**: ~3 seconds
- **Languages**: 100% Go

### Architecture

```
24 .go files across 8 packages
├── cmd/          7 files — command layer (thin)
├── internal/
│   ├── config/   2 files — XDG config management
│   ├── store/    1 file  — SQLite with WAL
│   ├── todo/     2 files — todo domain + tests
│   ├── ui/       2 files — theme + print helpers
│   └── version/  1 file  — build metadata
├── docs/         3 files — vision, decisions, status
└── scripts/      1 file  — install.sh
```

## Next Up: Phase 2

- [ ] `mine ai` — AI provider integrations (Claude, OpenAI, Ollama)
- [ ] `mine craft` — More scaffolding recipes (Rust, Docker, CI/CD)
- [ ] `mine dig` — TUI timer with bubbletea (replace printf)
- [ ] `mine stash` — Git-backed version history, restore, sync
- [ ] `mine shell` — Custom function injection, prompt integration
- [ ] Interactive TUI for `mine todo` using bubbletea

## Phase 3

- [ ] `mine vault` — Encrypted secrets store (age encryption)
- [ ] `mine grow` — Career goal tracking, learning streaks, skill radar
- [ ] `mine dash` — Full TUI dashboard with bubbletea
- [ ] Plugin system for community extensions
- [ ] Homebrew formula, AUR package, Nix flake
