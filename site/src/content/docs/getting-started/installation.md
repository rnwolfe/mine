---
title: Installation
description: How to install mine on your system
---

## Quick Install (Recommended)

```bash
curl -fsSL https://mine.rwolfe.io/install | bash
```

This downloads the latest release binary for your OS/arch and installs it to `~/.local/bin/`.

To inspect the script before running:

```bash
curl -fsSL https://mine.rwolfe.io/install -o install.sh
less install.sh
bash install.sh
```

## Build from Source

Requires Go 1.25+.

```bash
go install github.com/rnwolfe/mine@latest
```

Or clone and build:

```bash
git clone https://github.com/rnwolfe/mine.git
cd mine
make build
make install
```

## Verify Installation

```bash
mine version
```

## First-Time Setup

After installation, run the guided setup:

```bash
mine init
```

This will:
1. Ask for your name (auto-detected from git config)
2. Create config at `~/.config/mine/config.toml`
3. Create database at `~/.local/share/mine/mine.db`
4. Detect your shell for completions

## Shell Completions

Generate tab completions for your shell:

```bash
mine shell completions zsh    # for zsh
mine shell completions bash   # for bash
mine shell completions fish   # for fish
```

Follow the printed instructions to source them.

## Uninstall

```bash
rm -f "$(command -v mine)"
rm -rf ~/.config/mine
rm -rf ~/.local/share/mine
rm -rf ~/.cache/mine
```
