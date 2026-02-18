---
title: mine env
description: Encrypted per-project environment profiles with safe display, export, and injection
---

Manage per-project environment profiles with encrypted-at-rest storage outside the repository.
Values are masked by default in CLI output.

## Overview

`mine env` stores profiles under your local data directory and tracks each project's active
profile in SQLite. Profiles are named (for example: `local`, `staging`, `prod`).

- Storage path: `$XDG_DATA_HOME/mine/envs/` (default `~/.local/share/mine/envs/`)
- Profile files: encrypted `.age` files keyed by project path
- Active profile tracking: SQLite `env_projects` table

`mine env` reads passphrases from:

1. `MINE_ENV_PASSPHRASE`
2. `MINE_VAULT_PASSPHRASE`
3. Interactive prompt (TTY only)

In non-interactive mode without passphrase env vars set, commands fail with a clear error.

## Commands

### mine env

Show the current project's active profile with masked values.

```bash
mine env
```

### mine env show [profile]

Show variables for the active profile (default) or a named profile.

```bash
mine env show
mine env show staging
mine env show staging --reveal
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--reveal` | `false` | Show raw values instead of masked values |

### mine env set KEY=VALUE | KEY

Set a variable in the active profile.

```bash
mine env set API_URL=https://api.example.com
mine env set API_TOKEN     # prompts for value when interactive
printf '%s\n' "$TOKEN" | mine env set API_TOKEN
```

If you pass only `KEY`, the value is read securely from TTY input (no echo) or from stdin.

### mine env unset KEY

Remove a variable from the active profile.

```bash
mine env unset API_TOKEN
```

### mine env diff <profile-a> <profile-b>

Diff two profiles by key, showing added/removed/changed keys only.

```bash
mine env diff local staging
```

Values are not printed in diff output.

### mine env switch <profile>

Switch the active profile for the current project.

```bash
mine env switch staging
```

The target profile must already exist.

### mine env export

Emit shell export commands for the active profile.

```bash
mine env export
mine env export --shell fish
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--shell` | `posix` | Export syntax: `posix` or `fish` |

### mine env template

Generate `.env.example`-style output using keys only (no values).

```bash
mine env template > .env.example
```

### mine env inject -- <command> [args...]

Run a command with the active profile variables injected in that subprocess environment.

```bash
mine env inject -- go test ./...
mine env inject -- env | rg API_
```

## Security Notes

- Profile files are encrypted at rest using age passphrase encryption.
- Output is masked by default; use `--reveal` only when needed.
- Interactive `set KEY` hides input to avoid terminal echo/history leaks.
- Profile data is stored outside your git repository tree.

## Shell Integration

Use `menv` from `mine shell init` to load your active profile into the current shell:

```bash
eval "$(mine shell init)"
menv
```

On fish, `menv` uses fish-specific export format automatically.

