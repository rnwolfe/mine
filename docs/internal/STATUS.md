# mine — Project Status

## v0.3.0-alpha.1 — Phase 2: Growth (Complete)

Phase 1 (Foundation) is complete. Phase 2 is now fully complete — all planned
features shipped. Two Phase 3 features (vault, plugin system) shipped early due
to synergy needs (vault for AI key storage, plugins for extensibility).

### Phase 1: Foundation — Complete

- [x] **Core framework** — Cobra CLI, Go module, build system (Makefile)
- [x] **Theme system** — Lipgloss-based colors, icons, styled output (`internal/ui`)
- [x] **Config** (`mine config`) — TOML config, XDG-compliant paths, get/set/list subcommands
- [x] **SQLite store** — WAL mode, auto-migrations, high-perf pragmas (`internal/store`)
- [x] **Dashboard** (`mine`) — At-a-glance status with greeting, todos, tips
- [x] **Init** (`mine init`) — Guided onboarding, git name detection, environment discovery + auto-registration, shell helper RC file integration, idempotent re-run with `--reset` flag
- [x] **Todos** (`mine todo`) — Add, complete, delete, edit, list with priority/due/tags
- [x] **Stash** (`mine stash`) — Dotfile tracking, diff detection, git-backed history, restore, sync
- [x] **Version** (`mine version`) — Build-time version injection via ldflags
- [x] **Install script** — curl | bash installer for releases

### Phase 2: Growth — Complete

- [x] **Craft** (`mine craft`) — Project scaffolding with embedded recipe engine (Go/Node/Python/Git)
- [x] **Dig** (`mine dig`) — Pomodoro focus timer with streaks and stats
- [x] **Shell** (`mine shell`) — Completions (bash/zsh/fish), git helper functions, `menv` helper
- [x] **AI** (`mine ai`) — Multi-provider integrations (Claude, OpenAI, Gemini, OpenRouter), system instructions, terminal markdown rendering, vault-backed API keys
- [x] **Env** (`mine env`) — Encrypted per-project environment profiles (age encryption, show/set/unset/diff/switch/export/template/inject/edit)
- [x] **Proj** (`mine proj`) — Project registry, context switching, shell helpers (`p`, `pp`)
- [x] **Git** (`mine git`) — Branch sweep, PR generation, changelog, wip/unwip
- [x] **Tmux** (`mine tmux`) — Session management, named layouts, layout preview/delete, project integration, window management, rename, TUI picker
- [x] **SSH** (`mine ssh`) — SSH connection management
- [x] **Hook** (`mine hook`) — Four-stage hook pipeline (prevalidate/preexec/postexec/notify), user-local hooks
- [x] **TUI picker** — Reusable Bubbletea-based fuzzy-search picker (`internal/tui`)
- [x] **Tips** (`mine tips`) — Contextual tips system
- [x] **Doctor** (`mine doctor`) — Health check and diagnostics
- [x] **Meta** (`mine meta`) — Interact with mine-as-a-product (feature requests, bug reports, contribution workflows)
- [x] **About** (`mine about`) — About/build information
- [x] **Status** (`mine status`) — Mine status for shell prompt integration (JSON, compact prompt segment)
- [x] **Contrib** (`mine contrib`) — Community contribution helpers
- [x] **Agents** (`mine agents`) — Unified coding agent configuration manager (status, diff, sync, restore, project init/link, git versioning, adopt, add, list)
- [x] **Todo evolution** — Project-scoped todos (`--project`, cwd resolution), scheduling buckets (today/soon/later/someday), urgency sort (`mine todo next`), timestamped notes (`mine todo note`), dig focus integration (`mine dig --todo`), completion stats (`mine todo stats`), recurring todos (`--every` flag, auto-spawn on completion)
- [x] **Recurring tasks** — Auto-respawn on completion with `--every` flag

### Phase 3: Polish — Partially Started

- [x] **Vault** (`mine vault`) — Encrypted secrets store (age encryption, passphrase-based, system keychain persistence, AI key integration)
- [x] **Plugin system** (`mine plugin`) — Manifest, lifecycle, runtime, search, community plugin protocol
- [ ] **Grow** (`mine grow`) — Career growth tracking, learning streaks, skill radar
- [ ] **Dash** (`mine dash`) — Full TUI dashboard with bubbletea
- [ ] **Package distribution** — Homebrew formula, AUR package, Nix flake

### Infrastructure Shipped

- [x] **Documentation site** — Astro Starlight on Vercel (mine.rwolfe.io), auto-deploys
- [x] **Autodev pipeline** — Autonomous GitHub Actions implementation loop (dispatch/implement/review-fix/audit)
- [x] **Claude Code skills** — product, autodev, release, brainstorm, sweep-issues, refine-issue, draft-issue, personality-audit, autodev-audit
- [x] **CLI personality pass** — Consistent voice, warmth, and celebration across all output
- [x] **Analytics** (`internal/analytics`) — Anonymous usage telemetry (opt-in), routed through Vercel Edge Function ingest backend

### Binary Stats

- **Size**: ~21 MB (unstripped), ~7.6 MB (stripped release)
- **Deps**: 0 runtime (single static binary)
- **Build time**: ~3 seconds
- **Languages**: 100% Go
- **Tests**: 967 passing across 26 packages

### Architecture

```
184 .go files across 26 packages
├── cmd/            59 files — command layer (thin)
├── internal/
│   ├── agents/     24 files — unified coding agent configuration manager
│   ├── ai/         10 files — multi-provider AI integrations
│   ├── analytics/   4 files — anonymous usage telemetry
│   ├── config/      4 files — XDG config management
│   ├── contrib/     2 files — community contribution helpers
│   ├── craft/       3 files — recipe-driven scaffolding engine
│   ├── env/         2 files — encrypted per-project env profiles
│   ├── git/         2 files — git helpers (sweep, pr, changelog)
│   ├── gitutil/     1 file  — git utility helpers
│   ├── hook/        7 files — four-stage hook pipeline
│   ├── meta/        2 files — feature/bug report formatting helpers
│   ├── plugin/      7 files — plugin system (manifest, lifecycle, runtime)
│   ├── proj/        2 files — project registry + context switching
│   ├── shell/       5 files — completions, functions, shell init
│   ├── ssh/         6 files — SSH connection management
│   ├── stash/       5 files — dotfile tracking + git-backed history
│   ├── store/       2 files — SQLite with WAL
│   ├── tips/        2 files — contextual tips system
│   ├── tmux/        6 files — session + layout management
│   ├── todo/        6 files — todo domain logic (stats, notes, scheduling)
│   ├── tui/         8 files — reusable fuzzy-search picker
│   ├── ui/          5 files — theme, styles, print helpers
│   ├── vault/       7 files — age-encrypted secrets store
│   └── version/     2 files — build metadata
├── site/           Astro Starlight documentation site
├── docs/           Internal specs, plans, decisions
└── scripts/        Build, install, release, autodev helpers
```

## Remaining Roadmap

### Phase 3 — Not Started

- [ ] `mine grow` — Career growth tracking, learning streaks
- [ ] `mine dash` — Full TUI dashboard
- [ ] Package distribution — Homebrew, AUR, Nix

### Open Enhancements

- [ ] Test coverage improvement (#15)
