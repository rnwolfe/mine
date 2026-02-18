---
title: Environment Profiles
description: Encrypted, per-project environment variable profiles with safe display, export, and subprocess injection
---

Stop committing secrets or juggling `.env` files by hand. `mine env` stores per-project environment profiles encrypted at rest, outside your repository, with named profiles for different stages (local, dev, staging, prod — whatever you name them).

## Key Capabilities

- **Encrypted at rest** — profiles stored as age-encrypted files under `~/.local/share/mine/envs/`, never in your repo
- **Named profiles** — create as many profiles as you need (`local`, `staging`, `prod`, etc.)
- **Masked by default** — values are masked in CLI output; reveal only when you need to
- **Safe `set`** — read values from stdin or TTY prompt to keep them out of shell history
- **Shell export** — emit `export` statements (POSIX or fish) and load them into your session with `menv`
- **Subprocess injection** — run any command with profile vars in its environment, without exporting to your shell
- **Profile diff** — compare two profiles by key to see what's added, removed, or changed

## Quick Example

```bash
# Set variables (value read from prompt — never hits shell history)
mine env set API_URL=https://api.example.com
mine env set API_TOKEN

# Show active profile (values masked by default)
mine env

# Switch to staging profile
mine env switch staging

# Load active profile into your current shell
menv

# Run a command with profile vars injected (without polluting your shell)
mine env inject -- go test ./...

# Generate a .env.example for your repo
mine env template > .env.example
```

## How It Works

Each project gets an isolated directory under `~/.local/share/mine/envs/`, named by a SHA-256 hash of the project path. Inside that directory, each profile is a single age-encrypted file (`staging.age`, `prod.age`, etc.). The active profile for each project is tracked in SQLite.

All operations require a passphrase — provided via `MINE_ENV_PASSPHRASE`, `MINE_VAULT_PASSPHRASE`, or an interactive prompt. The passphrase is never written to disk.

`mine env inject -- <cmd>` merges profile variables into the subprocess environment (profile vars override any inherited values), so you can run builds, tests, or scripts with the right secrets without leaking them into your shell.

Add `eval "$(mine shell init)"` to your shell config and use `menv` as a one-word shortcut to load your active profile at any time.

## Learn More

See the [command reference](/commands/env/) for all subcommands, flags, and security details.
