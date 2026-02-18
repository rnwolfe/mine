---
title: Secrets Vault
description: Age-encrypted secret storage with clipboard integration and AI key management
---

Keep API keys and secrets off your disk in plaintext. `mine vault` stores secrets encrypted at rest using [age encryption](https://age-encryption.org/), accessible via a passphrase you never have to store anywhere.

## Key Capabilities

- **Age encryption** — secrets encrypted with scrypt key derivation, never written to disk in plaintext
- **Dot-notation keys** — organize secrets with namespaced keys (`ai.claude.api_key`, `db.prod.password`)
- **Clipboard integration** — copy secrets directly to clipboard without printing (`--copy`)
- **Export/import** — back up and restore the vault while keeping it encrypted
- **AI integration** — AI provider keys stored in the vault, used automatically by `mine ai`

## Quick Example

```bash
# Store a secret
mine vault set github.token ghp_abc123...

# Retrieve it
mine vault get github.token

# Copy to clipboard without printing
mine vault get github.token --copy

# List all keys (values are never shown)
mine vault list
```

## How It Works

The vault is a single age-encrypted file at `~/.local/share/mine/vault.age`. Every operation requires your passphrase — provided interactively or via the `MINE_VAULT_PASSPHRASE` environment variable for scripts. The passphrase is held in memory only during the operation, never written to disk.

AI commands (`mine ai ask`, `mine ai review`) automatically check the vault for API keys, so you can configure once with `mine ai config --provider claude --key sk-ant-...` and the key is stored encrypted. Environment variables take precedence if set.

The vault file is written atomically (temp file, fsync, rename) with `0600` permissions. If you forget your passphrase, the vault cannot be recovered — keep a backup with `mine vault export`.

## Learn More

See the [command reference](/commands/vault/) for all subcommands, security details, and error handling.
