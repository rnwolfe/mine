---
title: Shell Integration
description: Tab completions, aliases, shell functions, and prompt integration
---

Make mine feel native in your shell. `mine shell` generates tab completions for bash/zsh/fish, provides recommended aliases, and adds helper functions for git, SSH, and environment workflows.

## Key Capabilities

- **Tab completions** — auto-detect or specify your shell (`bash`, `zsh`, `fish`)
- **Aliases** — one-letter shortcuts for common commands (`m`, `mt`, `mg`, `mx`, etc.)
- **Git functions** — `gc`, `gca`, `gp`, `gpl`, `gsw` for common git one-liners
- **SSH functions** — `sc`, `scp2`, `stun`, `skey` for connections, tunnels, and key management
- **`menv`** — load your active `mine env` profile into the current shell session with one word
- **Shell init** — `eval "$(mine shell init)"` loads all functions into your session

## Quick Example

```bash
# Generate completions for your shell
mine shell completions zsh

# See all recommended aliases
mine shell aliases

# Load shell functions into your session
eval "$(mine shell init)"

# Load active env profile into current shell
menv
```

## How It Works

Add `eval "$(mine shell init)"` to your shell config (`.zshrc`, `.bashrc`, or `config.fish`) and you get the full set of helper functions on every new shell. Tab completions are generated separately — run `mine shell completions` and follow the printed instructions to source them.

The aliases are optional — `mine shell aliases` prints them so you can copy the ones you want. The full set gives you `m` for the dashboard, `mt` for todos, `mg` for git, `mx` for tmux, and more.

`menv` is a single-word shortcut for `eval "$(mine env export)"` that loads your active project profile into the current shell. Switch profiles with `mine env switch staging`, then run `menv` to apply them. Works in bash, zsh, and fish.

## Learn More

See the [command reference](/commands/shell/) for completion setup instructions, the full alias list, and all shell functions.
