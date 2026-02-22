---
title: mine vault
description: Encrypted secrets vault using age encryption
---

Store and retrieve secrets encrypted at rest using [age encryption](https://age-encryption.org/).
Secrets live in a single encrypted file at `~/.local/share/mine/vault.age`.

## Passphrase

All vault operations require a passphrase. mine resolves it in this order:

1. **`MINE_VAULT_PASSPHRASE` env var** — always wins, ideal for scripts
2. **OS keychain** — stored via `mine vault unlock`, retrieved transparently
3. **Interactive prompt** — secure input (no echo) when running in a terminal

The passphrase is never stored on disk in plaintext. By default it is only held in memory for the duration of each vault operation, but if you run `mine vault unlock` it may also be persisted securely in the OS keychain (encrypted/protected by the OS).

## Passphrase Management

### Store in OS Keychain

```bash
mine vault unlock
```

Prompts for the passphrase and stores it securely in the OS native keychain (macOS Keychain, GNOME Keyring on Linux via `secret-tool`). After unlocking, all vault commands work without prompting or setting env vars.

Platform behaviour:
- **macOS**: Uses the built-in `security` CLI — no extra dependencies.
- **Linux**: Requires `secret-tool` (part of `libsecret-tools`). If `secret-tool` is not installed, `unlock` returns an error indicating keychain is not available; use `MINE_VAULT_PASSPHRASE` instead.
- **Other platforms**: Returns an informational error; use `MINE_VAULT_PASSPHRASE` instead.

### Remove from OS Keychain

```bash
mine vault lock
```

Removes the stored passphrase from the OS keychain. Future vault commands will prompt again.

If no passphrase is stored, `lock` exits successfully with an informational message.

## Store a Secret

```bash
mine vault set <key> <value>
```

Keys use dot-notation namespacing by convention:

```bash
mine vault set ai.claude.api_key sk-ant-...
mine vault set ai.openai.api_key sk-...
mine vault set db.production.password hunter2
mine vault set github.token ghp_...
```

If the key already exists, the value is overwritten.

## Retrieve a Secret

```bash
mine vault get <key>
```

Prints the secret to stdout:

```bash
mine vault get ai.claude.api_key
# sk-ant-...
```

### Copy to Clipboard

```bash
mine vault get <key> --copy
```

Copies the secret to the system clipboard without printing it. Requires `xclip` or `xsel` (Linux), `pbcopy` (macOS), or `wl-copy` (Wayland).

## List Secret Keys

```bash
mine vault list
```

Lists all stored secret keys. Values are **never** shown.

## Delete a Secret

```bash
mine vault rm <key>
```

Permanently removes a secret from the vault.

## Export for Backup

```bash
mine vault export
mine vault export --output vault-backup.age
```

Writes the encrypted vault blob to stdout or a file. The export is still age-encrypted — safe to store or transfer.

## Import a Backup

```bash
mine vault import vault-backup.age
```

Replaces the current vault with the contents of the backup file. The import must be a valid age-encrypted vault created by `mine vault export` using the same passphrase.

:::caution
Import **replaces** the existing vault entirely. There is no merge. Back up your current vault before importing.
:::

## Subcommand Reference

| Subcommand | Description |
|------------|-------------|
| `set <key> <value>` | Store or update a secret |
| `get <key>` | Retrieve a secret (use `--copy` to copy to clipboard) |
| `list` | List all stored secret keys |
| `rm <key>` | Permanently delete a secret |
| `export` | Export encrypted vault for backup |
| `import <file>` | Restore vault from a backup |
| `unlock` | Store passphrase in OS keychain |
| `lock` | Remove passphrase from OS keychain |

## AI Key Integration

You can store AI provider API keys in the vault in two ways:

**Option 1: Use `mine ai config` (stores key and sets provider in one step)**

```bash
mine ai config --provider claude --key sk-ant-...
# key stored in vault, provider set as default
```

**Option 2: Store manually, then set the provider**

```bash
mine vault set ai.claude.api_key sk-ant-...
mine ai config --provider claude
# provider set as default; key is read from vault when using AI commands
```

When AI commands run (e.g. `mine ai ask`), they check for keys in this order:
1. Standard environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.)
2. Vault (requires `MINE_VAULT_PASSPHRASE` or interactive prompt)

## Security Notes

- Secrets are encrypted with [age](https://age-encryption.org/) using passphrase-based scrypt key derivation.
- Plaintext values are **never** written to disk at any point.
- The vault file is written atomically (temp file → fsync → rename) to prevent corruption on crash.
- The vault file permissions are `0600` (owner read/write only).
- If you forget your passphrase, **the vault cannot be recovered**. Keep a backup.
- Wrong passphrase returns a non-zero exit code with an explicit error — there is no silent fallback.
- A corrupted vault file returns a non-zero exit code with remediation guidance — there is no auto-repair.

## Error Handling

| Situation | Behavior |
|-----------|----------|
| Wrong passphrase | Non-zero exit, explicit error with hint |
| Corrupted vault file | Non-zero exit, explicit error with hint |
| Missing key | Non-zero exit, key name in error |
| Empty vault on export | Non-zero exit, instructive error |
| Import with wrong passphrase | Non-zero exit, explicit error |
| `vault unlock` on non-TTY | Non-zero exit, "requires an interactive terminal" |
| `vault unlock` — keychain unavailable | Non-zero exit, error with `MINE_VAULT_PASSPHRASE` hint |
| `vault lock` — nothing stored | Zero exit, informational message (not an error) |
| `vault lock` — keychain unavailable | Non-zero exit, error with `MINE_VAULT_PASSPHRASE` hint |

## Vault File Location

`~/.local/share/mine/vault.age` (XDG `$XDG_DATA_HOME/mine/vault.age`)

Override the data directory with the `XDG_DATA_HOME` environment variable.
