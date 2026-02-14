# mine — Operating Manual

> This is the agentic knowledge base for the `mine` CLI tool. It captures architecture
> decisions, patterns, lessons learned, and development practices. It is the source of
> truth for how to work on this project. Not user-facing documentation.

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
make test         # Run all tests (with race detector)
make cover        # Coverage report
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
├── CONTRIBUTING.md  # How to contribute (root-level, standard OSS)
├── CHANGELOG.md     # Release history (root-level, standard OSS)
├── LICENSE          # MIT license
├── README.md        # Project overview and quick start
├── CLAUDE.md        # THIS FILE — agentic knowledge base
├── cmd/             # Cobra command definitions (thin orchestration layer)
├── internal/        # Core domain logic (NOT exported)
│   ├── config/      # XDG config management
│   ├── store/       # SQLite data layer
│   ├── ui/          # Theme, styles, print helpers
│   ├── todo/        # Todo domain
│   ├── version/     # Build-time version info
│   └── ...          # New domains go here
├── docs/            # User-facing documentation (install, commands, architecture)
│   └── internal/    # Agentic docs (vision, decisions, status — NOT user-facing)
├── scripts/         # Install, release, CI helpers
├── site/            # Landing page (Vercel, auto-deploys)
└── .github/         # CI/CD workflows, CODEOWNERS, PR template
```

Rules:
- `cmd/` files are thin — parse args, call domain logic, format output
- `internal/` packages own their domain — they don't import each other unnecessarily
- Keep files under 500 lines
- Tests live next to the code they test (`_test.go` suffix)
- Core OSS docs (README, CONTRIBUTING, CHANGELOG, LICENSE) live at repo root
- User-facing docs live in `docs/`
- Agentic/internal docs live in `docs/internal/`

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

## Development Workflow

- **main is sacred.** All changes go through PRs. No direct pushes.
- Branch naming: `feat/`, `fix/`, `chore/`, `docs/` prefixes
- PRs require CI passing (`test` job). Copilot provides automated review.
- Human merges PRs after reviewing.
- CODEOWNERS: `@rnwolfe` reviews everything
- Site: https://mine.rwolfe.io (Vercel)
- Repo: https://github.com/rnwolfe/mine

## Release Process

- Tags trigger releases via GoReleaser (GitHub Actions)
- Format: `vMAJOR.MINOR.PATCH` (semver)
- CHANGELOG.md updated before tagging
- Binaries: linux/darwin x amd64/arm64

## Backlog Management

Feature backlog is tracked via GitHub Issues with labels:
- `feature` — new feature requests
- `enhancement` — improvements to existing features
- `phase/1`, `phase/2`, `phase/3` — roadmap phase
- `good-first-issue` — approachable for new contributors
- `spec` — has a spec document in `docs/internal/specs/`

Workflow:
1. Features start as GitHub Issues with clear acceptance criteria
2. Complex features get a spec doc in `docs/internal/specs/`
3. When ready to implement, create a branch from the issue
4. Issues reference the spec; PRs reference the issue

## Lessons Learned

### L-001: Git config name parsing
Git config values may be quoted (`name = "Ryan Wolfe"`). Always strip quotes
when parsing gitconfig values. Fixed in `cmd/init.go:gitUserName()`.

### L-002: TOML encoding of pre-quoted strings
If a value already contains quotes, TOML encoder will double-escape them.
Always clean input before saving to config.

### L-003: Working directory drift
When using `cd` in Bash tool calls (e.g., `cd site && vercel deploy`), the CWD
persists across subsequent calls. Always use absolute paths or explicitly `cd`
back to project root for subsequent commands.

### L-004: Vercel project naming
When deploying from a subdirectory (`site/`), Vercel uses the directory name
as the project name. Deploy from project root or use `--name` flag to control.

### L-005: GitHub Rulesets API schema sensitivity
The rulesets API (`POST /repos/{owner}/{repo}/rulesets`) is very picky about
the `rules[].parameters` shape. The `pull_request` type requires ALL five boolean
params to be present. When in doubt, create the ruleset in UI first, export it,
and use that JSON as the template.

### L-006: Self-approval impossible on GitHub
When pushing PRs via `gh` under your own token, you can't approve your own PRs.
Branch protection requiring approvals blocks the author. Solution: use CI checks
as the gate and Copilot for automated review, human merges manually.

### L-007: Third-party scaffolding cleanup
The claude-flow CLI scaffolded 355 files (.claude/agents/, .claude-flow/, .swarm/,
.mcp.json, hooks in settings.json) as part of initial setup. These were generic
templates unrelated to the Go project. Lesson: audit scaffolding tools before
committing. Remove attribution settings (`settings.json.attribution`) immediately
to prevent unwanted co-author credits in git history.

### L-008: Copilot review catches real issues
Copilot code review found 7 legitimate issues on first PR: bc dependency in CI,
unchecked errors in tests, duplicated test setup, unsafe `rm $(which ...)`,
missing curl safety note, and doc/code mismatch. Treat it as a real reviewer.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/root.go` | Dashboard, command registration |
| `cmd/todo.go` | Todo CRUD commands |
| `internal/ui/theme.go` | Colors, icons, style constants |
| `internal/store/store.go` | DB connection, migrations |
| `internal/todo/todo.go` | Todo domain logic + queries |
| `internal/config/config.go` | Config load/save, XDG paths |
