---
title: mine env
description: Encrypted per-project environment profiles with safe display, export, and injection
---

Manage per-project environment profiles encrypted at rest, stored outside your repository.
Values are masked by default in CLI output; reveal is always explicit.

## Passphrase

All `mine env` operations require a passphrase. Provide it via:

- **Environment variable** (recommended for scripts): `MINE_ENV_PASSPHRASE=<passphrase>` or `MINE_VAULT_PASSPHRASE=<passphrase>`
- **Interactive prompt**: mine will prompt securely (no echo) when running in a terminal

The passphrase is never stored anywhere — it is only held in memory during the operation. In non-interactive mode without a passphrase env var, commands fail with a clear error.

## Show Active Profile

```bash
mine env
```

Shows the current project's active profile with values masked. Equivalent to `mine env show`.

## Show a Profile

```bash
mine env show
mine env show staging
mine env show staging --reveal
```

Shows variables for the active profile (default) or a named profile. Values are masked unless `--reveal` is passed.

| Flag | Default | Description |
|------|---------|-------------|
| `--reveal` | `false` | Print raw values instead of masked values |

## Set a Variable

```bash
mine env set API_URL=https://api.example.com
mine env set API_TOKEN                          # prompts securely — no shell history
printf '%s\n' "$TOKEN" | mine env set API_TOKEN # read value from stdin
```

Sets a variable in the active profile. Pass `KEY=VALUE` inline or pass only `KEY` to read the value securely from TTY input (hidden) or from stdin. Using the prompt or stdin keeps secrets out of shell history.

If the key already exists, the value is overwritten.

## Unset a Variable

```bash
mine env unset API_TOKEN
```

Removes a variable from the active profile permanently.

## Edit Profile in $EDITOR

```bash
mine env edit
mine env edit staging
```

Decrypts the active profile (or a named profile) to a secure temp file and opens it in `$EDITOR`. On clean editor exit, the file is re-encrypted and saved. The temp file is removed on all exit paths — success, editor error, or save error.

The temp file format is sorted `KEY=VALUE` lines (one per variable). Blank lines and lines starting with `#` are ignored on re-read.

| Behaviour | Detail |
|-----------|--------|
| `$EDITOR` not set | Non-zero exit with hint to set `EDITOR` or use `mine env set` |
| Editor exits non-zero | Changes discarded; original profile unchanged |
| Invalid key in edited file | Changes discarded; all invalid keys listed in error |
| Named profile does not exist | Non-zero exit — profile must exist before editing |
| Temp file permissions | `0600` — owner read/write only |

## Compare Profiles

```bash
mine env diff local staging
```

Shows keys that differ between two profiles: added, removed, and changed. Values are **never** shown in diff output.

## Switch Active Profile

```bash
mine env switch staging
```

Changes the active profile for the current project. The target profile must already exist.

## Export for Shell

```bash
mine env export
mine env export --shell fish
```

Emits shell export statements for the active profile. Use this with `eval` to load vars into your session, or pipe to a script.

| Flag | Default | Description |
|------|---------|-------------|
| `--shell` | `posix` | Export syntax: `posix` (bash/zsh) or `fish` |

Use the `menv` shell helper from `mine shell init` as a shortcut:

```bash
eval "$(mine shell init)"
menv
```

## Generate a Template

```bash
mine env template > .env.example
```

Emits `.env.example`-style output with keys only and empty values. Useful for documenting required variables in your repository without exposing any secrets.

## Inject into a Subprocess

```bash
mine env inject -- go test ./...
mine env inject -- env | grep API_
```

Runs a command with the active profile variables injected into the subprocess environment. Profile variables override any matching inherited environment variables. Your shell session is not affected.

## Shell Integration

Install the `menv` helper by adding `eval "$(mine shell init)"` to your shell config:

```bash
# ~/.zshrc or ~/.bashrc
eval "$(mine shell init)"
```

Then use `menv` to load your active profile at any time:

```bash
mine env switch staging
menv
echo "$API_URL"
```

On fish, `menv` automatically uses fish-compatible export syntax. In all shells, `menv` returns a non-zero exit code if export fails.

## Security Notes

- Profile files are encrypted at rest using [age](https://age-encryption.org/) with passphrase-based scrypt key derivation.
- Plaintext values are **never** written to disk at any point.
- Profile files are written atomically (temp file → rename) to prevent corruption.
- Profile file permissions are `0600` (owner read/write only).
- If you forget your passphrase, **the profile cannot be recovered**.
- Wrong passphrase returns a non-zero exit code with an explicit error — there is no silent fallback.
- A tampered or corrupted profile returns a non-zero exit code — there is no auto-repair.

## Error Handling

| Situation | Behavior |
|-----------|----------|
| Wrong passphrase | Non-zero exit, explicit error with hint |
| Corrupted profile | Non-zero exit, explicit error with hint |
| Missing profile on `switch` | Non-zero exit, profile name in error |
| No passphrase in non-interactive mode | Non-zero exit, instructive error |
| Invalid key name | Non-zero exit before any disk writes |
| `$EDITOR` not set (`env edit`) | Non-zero exit with fallback hint |
| Editor exits non-zero (`env edit`) | Non-zero exit; original profile unchanged |

## Storage Location

Profiles are stored at `$XDG_DATA_HOME/mine/envs/` (default `~/.local/share/mine/envs/`).

Each project gets a subdirectory named by `sha256(project_path)`. Profile files inside are named `<profile>.age`. The active profile per project is tracked in the mine SQLite database.

Override the data directory with the `XDG_DATA_HOME` environment variable.
