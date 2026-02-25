# Changelog

All notable changes to mine will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-02-24

### Added

- **Agents management** (`mine agents`) — Manage AI agent configurations as versioned,
  project-linked artifacts. Add, list, view status, diff changes, sync to projects,
  restore previous versions, and scaffold project-level agent directories with
  `mine agents add/list/status/diff/sync/restore/init/link`.
- **Project-scoped todos** (`mine todo --project`) — Filter and add todos scoped to a
  specific project directory; cwd-resolution makes it automatic when inside a registered
  project.
- **Todo scheduling buckets** — Triage todos into `today / soon / later / someday`
  buckets so daily priorities are always clear.
- **Todo urgency sort and next command** (`mine todo next`) — Surfaces the single most
  urgent todo based on priority, due date, and bucket — no more staring at a long list.
- **Todo notes** (`mine todo note`) — Attach timestamped annotations and running notes
  to any todo; perfect for capturing context mid-task.
- **Dig focus integration** (`mine dig --todo`) — Start a Pomodoro focus session tied
  directly to a specific todo; completion is tracked together.
- **Todo completion stats** (`mine todo stats`) — Velocity metrics: completions per day,
  current streak, and historical output at a glance.
- **Recurring todos** (`mine todo --every`) — Create todos that auto-respawn on a
  schedule (e.g. `--every week`) when completed; ideal for habits and recurring tasks.
- **Smart init with environment discovery** (`mine init`) — Detects installed tools
  (git, tmux, AI provider), shows a dynamic capability table on first run, and
  auto-registers the current git repo as a mine project.
- **Shell integration setup during init** — `mine init` now offers to write
  `eval "$(mine shell init)"` to your shell RC file, unlocking `p`, `pp`, and `menv`
  without any manual steps.

### Fixed

- **`mine init` re-run safety** — Running `mine init` on an existing install now shows
  current settings and prompts to update individual fields. Use `--reset` to
  intentionally start fresh.
- **TUI todo delete** — Pressing `d` in the interactive todo list now correctly removes
  locally-added todos that haven't been persisted yet.

### Changed

- CLI output personality refined — warmer messages, consistent styling helpers,
  celebrations on key milestones, and improved command descriptions across all commands.

## [0.3.0-alpha.1] - 2026-02-22

### Added

- **Config CLI** (`mine config list/get/set/unset/edit/path`) — Manage all settings from the terminal with type-aware validation and schema defaults. Eliminates manual TOML editing for common config changes. All subcommands are hook-wrapped for plugin observability.
- **Env edit** (`mine env edit`) — Open a profile's full environment in `$EDITOR` for bulk editing — faster than running `set` repeatedly.
- **First-run experience** (`mine tips`, `mine doctor`) — Guided onboarding screen on first launch, a `mine tips` command for contextual usage hints, and `mine doctor` to check system health and configuration.
- **Vault keychain passphrase** — Vault and env profile passphrases can now be persisted in the system keychain (macOS Keychain / libsecret). No more re-entering on each session.
- **Tmux window management** (`mine tmux window`) — Create, list, rename, and switch tmux windows within a session from the CLI.
- **Tmux project sessions** (`mine tmux project`) — Create or attach to a named tmux session scoped to a project directory in one command.
- **Tmux layout enhancements** — `mine tmux layout delete` removes saved layouts; `mine tmux new --layout <name>` auto-restores a layout when creating a session; `mine tmux layout preview` previews a layout's pane structure; bare `mine tmux layout` now opens an interactive TUI fuzzy-search picker.
- **Tmux rename** (`mine tmux rename`) — Rename the current or a named tmux session.
- **Stash restore `--force`** — Skip the confirmation prompt when restoring a stash entry (`mine stash restore --force`).
- **AI system instructions** — Configure persistent system instructions for AI providers via `mine config set ai.system_instructions`.
- **AI markdown rendering** — AI command output now renders markdown (bold, headers, code blocks) directly in the terminal.
- **Analytics ingest backend** — Usage pings are now routed through a Vercel Edge Function; raw client data never reaches third-party servers.

### Fixed

- Plugin manifest entry validation consolidated into `ValidateEntry()` — all install paths (local and registry) now apply the same rules consistently.
- Hook pipeline now fires correctly for every command — all Cobra handlers are wrapped with `hook.Wrap`, closing a gap where hooks silently didn't trigger on some subcommands.

### Changed

- CLI output personality updated across all commands — warmer greetings, clearer progress feedback, and small celebrations when you finish tasks.

## [0.2.0-alpha.1] - 2026-02-18

### Added

- **Plugin system** (`mine plugin`) — Full plugin architecture with hook pipeline, manifest validation, permission sandboxing, and GitHub-based plugin search/install
- **User-local hooks** — Auto-discovered scripts in `~/.config/mine/hooks/` with filename convention routing, transform/notify modes, and CLI management (`mine hook list/create/test`)
- **Tmux management** (`mine tmux`) — Session list/new/attach/kill with layout persistence and reusable TUI fuzzy-search picker
- **Project management** (`mine proj`) — Project registry with add/rm/list/open/scan/config, context switching, shell helpers (`p`, `pp`), and dashboard integration
- **Environment profiles** (`mine env`) — Per-project encrypted environment variables with age encryption, profile switching, diff, export, templates, and shell injection (`menv`)
- **Secrets vault** (`mine vault`) — Encrypted secrets storage with age encryption
- **Git workflow** (`mine git`) — Git workflow supercharger commands
- **SSH helpers** (`mine ssh`) — SSH config management and connection helpers
- **AI integrations** (`mine ai`) — AI provider integrations
- **Meta commands** (`mine meta`) — Feature request, bug report, and community contribution helpers
- **Interactive TUI** — Bubbletea-based fuzzy-search picker for todo and dig, with non-TTY fallback
- **Stash versioning** (`mine stash`) — Git-backed version history with commit/log/restore/sync
- **Craft recipes** — Extended recipe engine with Rust, Docker, and GitHub CI templates; data-driven via `embed.FS`
- **Shell functions** — Shell function injection with prompt integration and `--help` flag support
- **Version flag** — `mine version --short` for script-friendly output
- **Anonymous analytics** — Opt-out usage analytics
- **Documentation site** — Astro Starlight site at mine.rwolfe.io with feature guides and command reference
- **Autonomous dev pipeline** — GitHub Actions workflow for agent-driven issue implementation with phased review (Copilot + Claude)
- **Backlog curation skills** — Claude Code skills for brainstorm, sweep-issues, refine-issue, draft-issue, and personality-audit

### Fixed

- Shell injection prevention in review body passing
- Autodev pipeline robustness (PAT for downstream workflows, stale branch cleanup, stderr logging, max turns/timeouts)
- CI path filters so non-code PRs aren't blocked
- Community contribution command reliability
- Cloud setup robustness for gh and sqlite

## [0.1.0] - 2026-02-14

### Added

- **Dashboard** (`mine`) — Personal status at a glance with greeting, todo summary, and contextual tips
- **Init** (`mine init`) — Guided first-time setup with git name auto-detection
- **Todos** (`mine todo`) — Full task management with add/complete/delete/edit, priority levels (low/med/high/crit), due dates (natural language: "tomorrow", "next-week"), and tags
- **Stash** (`mine stash`) — Dotfile tracking with init/track/list/diff workflow
- **Craft** (`mine craft`) — Project scaffolding for git, Go, Node.js, and Python
- **Dig** (`mine dig`) — Pomodoro focus timer with progress bar, streak tracking, and stats
- **Shell** (`mine shell`) — Tab completions for bash/zsh/fish, recommended alias list
- **Config** (`mine config`) — View configuration and paths
- **Version** (`mine version`) — Build-time version info via ldflags
- XDG Base Directory compliant paths
- SQLite storage with WAL mode and automatic migrations
- TOML configuration
- Install script (`scripts/install.sh`)
- 10 unit tests covering todo and config domains
