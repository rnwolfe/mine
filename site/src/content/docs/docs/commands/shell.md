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
```

Add these to your `~/.zshrc`, `~/.bashrc`, or `~/.config/fish/config.fish`.

## Examples

```bash
# Generate completions for zsh
mine shell completions zsh

# View recommended aliases
mine shell aliases
```
