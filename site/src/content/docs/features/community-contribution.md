---
title: Community Contribution
description: AI-assisted contribution workflow — fork, clone, branch, and jump straight to coding on any GitHub repo
---

Turbo-start an AI-assisted contribution workflow for any GitHub repository. `mine contrib` handles fork/clone orchestration, issue selection, and workspace setup so you can focus on writing code, not plumbing.

**Requires:** `gh` CLI installed and authenticated (`gh auth login`).

## Key Capabilities

- **Issue selection** — prefers `agent-ready` labeled issues when present; falls back to all open issues
- **Fork/clone detection** — reuses existing forks, clones only what's needed
- **Branch naming** — creates a clean `issue-<N>-<slug>` branch automatically
- **Explicit opt-in** — always shows what will happen before touching your GitHub account
- **Tmux workspace** — optional two-pane workspace for agent + shell side-by-side
- **Quota-safe** — all actions use your own GitHub account; you're always shown the impact before proceeding
- **mine shortcut** — `mine meta contrib` targets the mine repo itself

## Quick Example

```bash
# Start a contribution flow for any repo
mine contrib --repo owner/name

# List candidate issues without starting the flow
mine contrib --repo owner/name --list

# Jump straight to a specific issue
mine contrib --repo owner/name --issue 42

# Open a two-pane tmux workspace
mine contrib --repo owner/name --tmux

# Shortcut for contributing to mine itself
mine meta contrib
```

## How It Works

Run `mine contrib --repo owner/name` and mine fetches open issues labeled `agent-ready` (if any exist) or all open issues. In TTY mode you get an interactive fuzzy picker — type to filter, Enter to select. After you confirm the opt-in prompt, mine forks the repo (or reuses your existing fork), clones it locally, adds the `upstream` remote pointing to the original repo, and checks out a branch named `issue-<N>-<title-slug>`. The workspace is ready the moment the command returns.

The `--tmux` flag splits a new tmux session into two panes — a shell in the clone directory and a spare pane for running an agent. If tmux is unavailable the flag is silently ignored so the command always works.

In non-TTY environments (scripts, CI) you must pass `--issue N` directly — the picker is not available.

## Learn More

See the [command reference](/commands/contrib/) for all flags and the full issue selection policy.
