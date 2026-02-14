# Changelog

All notable changes to mine will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
