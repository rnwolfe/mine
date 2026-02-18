---
title: mine shell
description: Shell integration and enhancements
---

Shell integration and enhancements including completions and aliases.

## Generate Completions

```bash
mine shell completions         # auto-detect shell
mine shell completions zsh
mine shell completions bash
mine shell completions fish
```

Follow the printed instructions to source them in your shell config.

## Show Recommended Aliases

```bash
mine shell aliases
```

Outputs:
```bash
alias m='mine'
alias mt='mine todo'
alias mta='mine todo add'
alias mtd='mine todo done'
alias md='mine dig'
alias mc='mine craft'
alias ms='mine stash'
alias mx='mine tmux'
alias mg='mine git'
```

Add these to your `~/.zshrc`, `~/.bashrc`, or `~/.config/fish/config.fish`.

## Git Shell Functions

The following git helper functions are included in `mine shell init`:

| Function | Description |
|----------|-------------|
| `gc <msg>` | `git commit -m` shorthand |
| `gca <msg>` | `git commit --amend -m` shorthand |
| `gp` | `git push` with upstream tracking |
| `gpl` | `git pull --rebase` |
| `gsw <branch>` | `git switch` shorthand |

## SSH Shell Functions

The following SSH helper functions are included in `mine shell init`:

| Function | Description |
|----------|-------------|
| `sc <alias>` | Quick connect: `ssh <alias>` |
| `scp2 <src> <dest>` | Resumable copy: `rsync -avzP --partial` over SSH |
| `stun <alias> <L:R>` | Quick tunnel shorthand |
| `skey [file]` | Copy default public key to clipboard |

All functions include `--help` for usage documentation and work in bash, zsh, and fish.

## Env Shell Functions

The following env helper function is included in `mine shell init`:

| Function | Description |
|----------|-------------|
| `menv` | Load the active `mine env` profile into your current shell session |

Implementation details:

- Bash/Zsh: evaluates `mine env export` output
- Fish: evaluates `mine env export --shell fish` output
- All variants return a non-zero exit code if export fails

Example:

```bash
eval "$(mine shell init)"
mine env switch staging
menv
echo "$API_URL"
```

## Project Shell Functions

The following project helper functions are included in `mine shell init`:

| Function | Description |
|----------|-------------|
| `p [name]` | Quick project switch. With no args, opens picker. |
| `pp` | Switch to the previously active project |

These wrappers call `mine proj` / `mine proj open --print-path` and perform the `cd` in your shell process.

## Examples

```bash
# Generate completions for zsh
mine shell completions zsh

# View recommended aliases
mine shell aliases

# Load active env profile into current shell
eval "$(mine shell init)"
menv
```
