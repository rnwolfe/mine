# mine — Operating Manual

> This is the knowledge base for the `mine` CLI tool. It captures architecture decisions,
> patterns, lessons learned, and development practices. It is the source of truth for how
> to work on this project.

## Project Overview

**mine** is a developer CLI supercharger — a single Go binary that replaces the sprawl of
productivity tools with one fast, delightful, opinionated companion.

- **Language**: Go 1.25+
- **CLI**: Cobra
- **TUI**: Lipgloss (+ Bubbletea for interactive views, coming)
- **Storage**: SQLite via modernc.org/sqlite (pure Go, no CGo)
- **Config**: TOML at `~/.config/mine/config.toml` (XDG-compliant)
- **Binary**: Single static binary, ~7.6 MB stripped

## Build & Test

```bash
make build        # Build to bin/mine
make test         # Run all tests
make lint         # go vet ./...
make dev ARGS="todo"  # Quick dev cycle
make install      # Install to PATH
```

- ALWAYS run `make test` after code changes
- ALWAYS run `make build` before committing
- NEVER commit if tests fail

## File Organization

```
mine/
├── cmd/           # Cobra command definitions (thin orchestration layer)
├── internal/      # Core domain logic (NOT exported)
│   ├── config/    # XDG config management
│   ├── store/     # SQLite data layer
│   ├── ui/        # Theme, styles, print helpers
│   ├── todo/      # Todo domain
│   ├── version/   # Build-time version info
│   └── ...        # New domains go here
├── docs/          # Vision, decisions, status, guides
├── scripts/       # Install, release, CI helpers
├── config/        # Default/example configurations
├── site/          # Landing page (Vercel)
└── .github/       # CI/CD workflows
```

Rules:
- `cmd/` files are thin — parse args, call domain logic, format output
- `internal/` packages own their domain — they don't import each other unnecessarily
- Keep files under 500 lines
- Tests live next to the code they test (`_test.go` suffix)

## Architecture Patterns

1. **Domain separation**: Each feature is a package under `internal/`
2. **Store pattern**: SQLite via `store.DB` wrapper — domains get `*sql.DB` via `db.Conn()`
3. **UI consistency**: All output through `internal/ui` helpers — never raw `fmt.Println`
4. **Config**: Single TOML file, loaded once, XDG-compliant paths
5. **Progressive migration**: Schema changes via `store.migrate()` auto-applied on open

## Design Principles

1. **Speed**: Every local command < 50ms. No spinners for local ops.
2. **Single binary**: No runtime dependencies. `curl | sh` install.
3. **Opinionated defaults**: Works out of the box. Escape hatches exist.
4. **Whimsical but competent**: Fun personality in messages. Serious about results.
5. **Local first**: Data stays on machine. Cloud features opt-in.
6. **XDG-compliant**: `~/.config/mine/`, `~/.local/share/mine/`, `~/.cache/mine/`

## Personality Guide

- Use emoji sparingly and consistently (see `ui/theme.go` icon constants)
- Greeting should feel like a friend, not a robot
- Tips should be actionable, not generic
- Error messages should say what went wrong AND what to do about it
- Celebrate small wins (completing a todo, finishing a focus session)
- Never be annoying or preachy

## Security Rules

- NEVER hardcode secrets or API keys
- NEVER commit .env files
- Validate all user input at system boundaries
- Sanitize file paths (prevent directory traversal)
- SQLite uses WAL mode with busy timeout (safe for concurrent reads)

## Release Process

- Tags trigger releases via GoReleaser (GitHub Actions)
- Format: `vMAJOR.MINOR.PATCH` (semver)
- CHANGELOG.md updated before tagging
- Binaries: linux/darwin x amd64/arm64

## Lessons Learned

### L-001: Git config name parsing
Git config values may be quoted (`name = "Ryan Wolfe"`). Always strip quotes
when parsing gitconfig values. Fixed in `cmd/init.go:gitUserName()`.

### L-002: TOML encoding of pre-quoted strings
If a value already contains quotes, TOML encoder will double-escape them.
Always clean input before saving to config.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/root.go` | Dashboard, command registration |
| `cmd/todo.go` | Todo CRUD commands |
| `internal/ui/theme.go` | Colors, icons, style constants |
| `internal/store/store.go` | DB connection, migrations |
| `internal/todo/todo.go` | Todo domain logic + queries |
| `internal/config/config.go` | Config load/save, XDG paths |
