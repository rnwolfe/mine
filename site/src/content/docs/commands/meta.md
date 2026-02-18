---
title: mine meta
description: Interact with mine-as-a-product — feature requests, bug reports, and contribution workflows
---

Commands for engaging with the mine project itself. Submit feedback, report bugs,
or jump straight into contributing.

## mine meta fr

Submit a feature request as a GitHub issue.

```bash
mine meta fr "Add dark mode to the dashboard"
mine meta fr                          # interactive prompts
mine meta fr --dry-run "My idea"      # preview without submitting
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--description` | `-d` | What the feature should do |
| `--use-case` | `-u` | Why you need this feature |
| `--dry-run` | | Preview the issue without submitting |

## mine meta bug

Report a bug as a GitHub issue. Auto-detects your OS, architecture, and mine version.

```bash
mine meta bug "Dashboard crashes on empty todo list"
mine meta bug                         # interactive prompts
mine meta bug --dry-run "My bug"      # preview without submitting
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--steps` | `-s` | Steps to reproduce |
| `--expected` | `-e` | Expected behavior |
| `--actual` | `-a` | Actual behavior |
| `--dry-run` | | Preview the issue without submitting |

## mine meta contrib

Shortcut for `mine contrib --repo rnwolfe/mine`. Turbo-starts an AI-assisted
contribution workflow for the mine project itself.

```bash
mine meta contrib                  # start contribution flow for mine
mine meta contrib --list           # list candidate issues
mine meta contrib --issue 16       # target a specific issue
mine meta contrib --tmux           # open a two-pane tmux workspace
```

**Flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--issue N` | `-i N` | Work on a specific issue directly |
| `--list` | | List candidate issues without starting the flow |
| `--tmux` | | Start a two-pane tmux workspace |

See [mine contrib](/commands/contrib) for the full contribution workflow documentation.
