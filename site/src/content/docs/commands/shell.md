---
title: mine shell
description: Shell integration and enhancements
---

Shell completions and recommended aliases.

## Generate completions

```bash
mine shell completions         # auto-detect shell
mine shell completions zsh
mine shell completions bash
mine shell completions fish
```

Follow the printed instructions to source them in your shell config.

## Show recommended aliases

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

Copy these to your `~/.zshrc`, `~/.bashrc`, or equivalent.
