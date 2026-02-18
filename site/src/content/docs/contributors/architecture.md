---
title: Architecture
description: Technical overview of mine's architecture and design patterns
---

mine is a single Go binary built with Cobra (CLI), Lipgloss (styling), and SQLite (storage). It follows a clean domain separation pattern where each feature is an independent package under `internal/`.

## Directory Structure

```mermaid
graph TD
    A[main.go] --> B[cmd/]
    B --> C[internal/]
    C --> D[config/]
    C --> E[store/]
    C --> F[todo/]
    C --> G[ui/]
    C --> H[stash/]
    C --> I[craft/]
    C --> J[hook/]
    C --> K[plugin/]
```

```
mine/
├── main.go                  # Entry point — calls cmd.Execute()
├── cmd/                     # Command definitions (Cobra)
│   ├── root.go              # Dashboard, command registration
│   ├── init.go              # First-time setup
│   ├── todo.go              # Todo CRUD subcommands
│   ├── stash.go             # Dotfile management
│   ├── craft.go             # Project scaffolding
│   ├── dig.go               # Focus timer
│   ├── shell.go             # Shell completions & aliases
│   ├── config.go            # Config display
│   └── version.go           # Version info
├── internal/                # Domain logic (not exported)
│   ├── config/              # XDG config management
│   │   ├── config.go        # Load, save, paths
│   │   └── config_test.go
│   ├── store/               # SQLite database
│   │   └── store.go         # Connection, migrations
│   ├── todo/                # Todo domain
│   │   ├── todo.go          # Models, queries, store
│   │   └── todo_test.go
│   ├── ui/                  # Terminal UI
│   │   ├── theme.go         # Colors, icons, styles
│   │   └── print.go         # Output helpers
│   └── version/             # Build metadata
│       └── version.go
├── site/                    # Documentation site (Astro Starlight)
├── scripts/                 # Build & install helpers
├── .github/                 # CI/CD workflows
├── .goreleaser.yaml         # Release configuration
├── Makefile                 # Build commands
└── CLAUDE.md                # Project knowledge base
```

## Design Patterns

### 1. Thin Command Layer

Files in `cmd/` are orchestration only. They:
- Parse arguments and flags
- Call domain logic in `internal/`
- Format output using `internal/ui`

They do **not** contain business logic or direct database queries.

```mermaid
sequenceDiagram
    User->>cmd: mine todo add "task"
    cmd->>internal/todo: CreateTodo(...)
    internal/todo->>store: INSERT query
    store-->>internal/todo: todo ID
    internal/todo-->>cmd: todo object
    cmd->>internal/ui: FormatTodo(...)
    internal/ui-->>User: ✓ Added todo #1
```

### 2. Domain Packages

Each feature owns its domain under `internal/`:
- `todo` owns the todo model, queries, and store interface
- `config` owns configuration loading and XDG path resolution
- `store` owns database connection, pragmas, and migrations
- `ui` owns all terminal styling and output formatting

Packages don't import each other unnecessarily. `store.DB` provides `*sql.DB` via `.Conn()`, and domain packages accept `*sql.DB` directly.

### 3. Progressive Migration

Schema changes are defined in `store.migrate()` and auto-applied on every `store.Open()`. This means:
- No migration CLI needed
- Database always matches expected schema
- New tables added with `CREATE TABLE IF NOT EXISTS`
- Safe for concurrent reads (WAL mode)

### 4. XDG Compliance

All file paths follow the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/):

| Purpose | Path | Env Override |
|---------|------|-------------|
| Config | `~/.config/mine/` | `$XDG_CONFIG_HOME` |
| Data | `~/.local/share/mine/` | `$XDG_DATA_HOME` |
| Cache | `~/.cache/mine/` | `$XDG_CACHE_HOME` |
| State | `~/.local/state/mine/` | `$XDG_STATE_HOME` |

### 5. Consistent UI

All output goes through `internal/ui` helpers:
- `ui.Ok()`, `ui.Err()`, `ui.Warn()` — semantic messages
- `ui.Kv()` — key-value pairs with consistent padding
- `ui.Header()` — section headers
- `ui.Tip()` — contextual tips
- `ui.Greet()` — personality-infused greetings

Never use raw `fmt.Println` in commands.

## Data Flow

```mermaid
flowchart LR
    A[User Input] --> B[Cobra cmd/]
    B --> C[Domain Logic internal/]
    C --> D[SQLite store/]
    C --> E[UI Formatting ui/]
    E --> F[Terminal Output]
```

## SQLite Configuration

The database uses performance-optimized pragmas:
- `journal_mode=WAL` — concurrent reads, single writer
- `synchronous=NORMAL` — safe with WAL
- `cache_size=-64000` — 64MB in-memory cache
- `foreign_keys=ON` — referential integrity
- `temp_store=MEMORY` — temp tables in RAM
- `busy_timeout=5000` — 5s retry on lock contention

## Build System

- `make build` — builds with ldflags for version injection
- `make test` — runs the Go test suite
- GoReleaser handles cross-compilation for releases
- CI runs vet, test (with coverage), build, and smoke test

## Plugin System

mine supports extensibility via plugins:

```mermaid
graph LR
    A[Command] --> B[Hook Pipeline]
    B --> C[prevalidate]
    C --> D[preexec]
    D --> E[Execute Command]
    E --> F[postexec]
    F --> G[notify]
```

Plugins are standalone binaries that communicate via JSON-over-stdin. See the [Plugin Protocol](/contributors/plugin-protocol/) for details.

## Contribution Workflow (`internal/contrib/`)

The `contrib` package orchestrates AI-assisted contribution workflows. It has no persistent
storage — all operations are transient GitHub API calls and local filesystem operations via
the `gh` CLI.

Key responsibilities:
- Repo slug validation (`ValidateRepo`)
- Issue fetching and selection policy (`FetchCandidateIssues`, `FetchIssue`)
- Fork detection and creation (`CheckForkState`, `EnsureFork`)
- Clone and branch setup (`CloneRepo`, `CreateBranch`, `BranchName`)

The `cmd/contrib.go` command wires these together with:
- An explicit opt-in confirmation prompt before any action
- A clear warning that actions use the user's own GitHub quota
- TTY-aware issue selection (interactive picker in TTY, `--issue` required otherwise)
- Optional `--tmux` flag for a two-pane workspace

`mine meta contrib` is a thin shortcut in `cmd/meta.go` that calls the same flow
with `rnwolfe/mine` as the target repo.
