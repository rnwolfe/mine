---
title: mine contrib
description: AI-assisted contribution workflow for any GitHub repo — fork, clone, branch, and get to work
---

Turbo-start an AI-assisted contribution workflow for any GitHub repository.
Handles fork/clone orchestration, issue selection, and workspace setup so you
can focus on writing code, not plumbing.

**Requires:** `gh` CLI installed and authenticated (`gh auth login`).

## Basic usage

```bash
mine contrib --repo owner/name
```

Opens an interactive issue picker, forks the repo (or reuses an existing fork),
clones it locally, creates a branch, and drops you in a ready-to-code workspace.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--repo owner/name` | | Target GitHub repository (required) |
| `--issue N` | `-i N` | Work on a specific issue directly |
| `--list` | | List candidate issues without starting the flow |
| `--tmux` | | Start a two-pane tmux workspace (agent + shell) |

## Examples

```bash
# Interactive: pick from agent-ready issues (or all open issues)
mine contrib --repo owner/name

# Target a specific issue directly
mine contrib --repo owner/name --issue 42

# List candidate issues first
mine contrib --repo owner/name --list

# Open a two-pane tmux workspace
mine contrib --repo owner/name --tmux

# Shortcut for contributing to mine itself
mine meta contrib
```

## Issue selection policy

1. If `--issue N` is provided, that issue is used directly.
2. If the repo has open issues labeled `agent-ready`, those are preferred.
3. Otherwise, all open issues are listed.
4. In TTY mode, an interactive fuzzy picker is shown.
5. In non-TTY mode (scripts, CI), `--issue` is required.

## Workspace setup

After confirming the opt-in prompt, the flow:

1. **Forks** the repo under your GitHub account (skipped if you already have one).
2. **Clones** your fork locally into `./<repo-name>/`.
3. **Adds** the upstream remote (`upstream`) pointing to the original repo.
4. **Creates** a branch named `issue-<N>-<slug>`.

If the local directory already exists, the command stops and tells you to remove
it or change directories before retrying.

## Quota warning

> All actions (fork, clone, branch creation) use your own GitHub account and API
> quota. You will be shown an explicit opt-in prompt before any action is taken.

## mine meta contrib

`mine meta contrib` is a shortcut that targets the mine repo itself:

```bash
mine meta contrib            # start contribution flow for rnwolfe/mine
mine meta contrib --list     # list candidate issues for rnwolfe/mine
mine meta contrib --issue 16 # target a specific issue
```
