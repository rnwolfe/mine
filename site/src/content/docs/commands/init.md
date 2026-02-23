---
title: mine init
description: Guided first-time setup and configuration update
---

Guided setup for mine. Creates config and data directories, detects your environment, and optionally registers the current directory as a mine project. Safe to re-run — if config already exists, it shows your current settings and offers to update them.

## Usage

```bash
mine init           # Fresh install or update existing config
mine init --reset   # Overwrite config from scratch
```

## Flags

| Flag | Description |
|------|-------------|
| `--reset` | Overwrite config from scratch (shows warning and asks for confirmation) |

## Fresh Install Behavior

On a fresh install (no existing config), `mine init` runs the full interactive wizard:

1. Auto-detects your name from `~/.gitconfig`
2. Configures AI provider (detects existing API keys or guides OpenRouter setup)
3. Offers to write `eval "$(mine shell init)"` to your RC file — enabling `p`, `pp`, and `menv`
4. Creates config at `~/.config/mine/config.toml`
5. Creates database at `~/.local/share/mine/mine.db`
6. Probes your environment to show a capability table
7. If run inside a git repo, offers to register it as a mine project

## Re-run Behavior (Existing Config)

When `mine init` detects an existing config, it shows your current settings and asks if you want to update them:

```
▸  mine is already set up.

  Your current configuration:
    Name     Ryan Wolfe
    AI       claude (claude-sonnet-4-5-20250929)
    Shell    /bin/zsh

  Update your configuration? (y/N)
```

- **N (default)**: exits immediately with no changes
- **Y**: runs the same prompts as fresh install, with each field pre-filled with your current value — pressing Enter keeps it unchanged

Re-init preserves these fields unconditionally (they are never shown in prompts):
- Analytics preference (`analytics.enabled`)
- Analytics installation ID
- Vault keys
- SQLite database
- Any config field not surfaced in prompts

After a successful re-init, the output says **"Configuration updated"** rather than "All set!".

## Reset Behavior (`--reset`)

`mine init --reset` provides a hard-reset path when you want to start from scratch:

```
  ⚠ This will overwrite your current configuration.

  Proceed? (y/N)
```

- **N (default)**: exits with no changes
- **Y**: runs the full fresh-install wizard, replacing the config file

What `--reset` replaces: the config file at `~/.config/mine/config.toml`

What `--reset` does NOT touch: analytics ID, vault keys, and the SQLite database (`mine.db`)

## Shell Integration Step

After the AI setup section, `mine init` shows:

```
  Shell Integration

  Adding this line to ~/.zshrc enables p, pp, and menv:

    eval "$(mine shell init)"

  Add it now? (Y/n)
```

- **Y (default)**: appends `eval "$(mine shell init)"` to your RC file and confirms the path
- **n**: prints the line so you can add it manually
- **Already present**: silently skipped — safe to re-run `mine init`
- **Unrecognized shell**: prints the line for manual addition; `mine init` still completes

Supported shells: `zsh` (`~/.zshrc`), `bash` (`~/.bashrc` / `~/.bash_profile`), `fish` (`~/.config/fish/config.fish`).

The appended content is exactly:

```
# added by mine
eval "$(mine shell init)"
```

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

## Example — Fresh Install

```
$ mine init
▸  Welcome to mine!

  Let's get you set up. This takes about 30 seconds.

  What should I call you? (Ryan Wolfe)

  AI Setup (optional)
  ...

  Shell Integration

  Adding this line to ~/.zshrc enables p, pp, and menv:

    eval "$(mine shell init)"

  Add it now? (Y/n)

✓ Added to ~/.zshrc. Restart your shell or run: source ~/.zshrc

✓ All set, Ryan! 🎉

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

## Example — Re-run to Update Name

```
$ mine init
▸  mine is already set up.

  Your current configuration:
    Name     Ryan Wolfe
    AI       claude (claude-sonnet-4-5-20250929)
    Shell    /bin/zsh

  Update your configuration? (y/N) y

  What should I call you? (Ryan Wolfe) Ryan W

  AI Setup (optional)
  ...

✓ Configuration updated, Ryan W!
```
