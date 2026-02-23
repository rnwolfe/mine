---
title: mine init
description: Guided first-time setup
---

Guided first-time setup. Creates config and data directories, detects your environment, and optionally registers the current directory as a mine project.

## Usage

```bash
mine init
```

## What It Does

1. Auto-detects your name from `~/.gitconfig`
2. Creates config at `~/.config/mine/config.toml`
3. Creates database at `~/.local/share/mine/mine.db`
4. Guides you through optional AI provider setup
5. Probes your environment to show a capability table
6. If run inside a git repo, offers to register it as a mine project

## Capability Table

After setup, `mine init` prints a dynamic table showing which features are ready to use based on what's installed and configured on your system:

```
  What you've got:

    ✓  todos          — mine todo add "ship it"
    ✓  stash          — mine stash add <url>
    ✓  env            — mine env init
    ✓  git            — mine git log
    ✓  tmux           — mine tmux new
    ✓  AI (claude)    — mine ai ask "explain this diff"
    ·  proj           — mine proj add <path>
```

- `✓` rows are ready — each shows a concrete command to try immediately
- `·` rows need setup — each shows a one-line hint for what to do next

The capability checks are:

| Capability | Ready when |
|------------|------------|
| todos      | Always ready |
| stash      | Always ready |
| env        | Always ready |
| git        | `git` binary found in `$PATH` |
| tmux       | `tmux` binary found in `$PATH` |
| AI         | An AI provider is configured (`mine ai config`) |
| proj       | Current directory was just registered as a project |

## Project Auto-Registration

When run inside a git repository, `mine init` prompts you to register the current directory as a mine project:

```
  Register /path/to/your/project as a mine project? (Y/n)
```

Answering `y` (or pressing Enter) calls `mine proj add` on the current directory. This makes it immediately available in `mine proj` and sets the dashboard's project context.

If you're not inside a git repo, this prompt is skipped silently.

## Example

```
$ mine init
▸  Welcome to mine!

  Let's get you set up. This takes about 30 seconds.

  What should I call you? (Ryan Wolfe)

  AI Setup (optional)
  ...

  ✓ All set!

  Created:
    Config  ~/.config/mine/config.toml
    Data    ~/.local/share/mine/mine.db

  Hey Ryan — you're ready to go. Type mine to see your dashboard.

  Register /home/ryan/projects/myapp as a mine project? (Y/n) y

  ✓  Registered project myapp

  What you've got:

    ✓  todos          — mine todo add "ship it"
    ✓  stash          — mine stash add <url>
    ✓  env            — mine env init
    ✓  git            — mine git log
    ·  tmux           — install tmux, then mine tmux new
    ✓  AI (claude)    — mine ai ask "explain this diff"
    ✓  proj           — mine proj list
```
