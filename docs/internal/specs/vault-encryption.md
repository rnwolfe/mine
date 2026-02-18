# Vault Encryption Spec

Internal design document for `mine vault` — encrypted local secrets storage.

## File Format

**Path**: `$XDG_DATA_HOME/mine/vault.age` (default: `~/.local/share/mine/vault.age`)

**Encoding**: age-armored (ASCII PEM-like) — text file, safe to `cat` and store in git (encrypted).

**Encryption**: `filippo.io/age` v1 with `ScryptRecipient`/`ScryptIdentity` (passphrase-based).

**Scrypt parameters**: age defaults — `N=2^18`, `r=8`, `p=1`. Intentionally slow for brute-force resistance.

**Plaintext payload**: JSON object (before encryption):

```json
{
  "secrets": {
    "ai.claude.api_key": "sk-ant-...",
    "ai.openai.api_key": "sk-..."
  }
}
```

## Key Naming Convention

Keys use dot-notation namespacing:

```
<domain>.<entity>.<attribute>
```

| Domain | Example key | Description |
|--------|-------------|-------------|
| `ai` | `ai.claude.api_key` | AI provider API keys |
| `db` | `db.production.password` | Database credentials |
| `github` | `github.token` | GitHub tokens |

AI provider keys follow the pattern `ai.<provider>.api_key` where `<provider>` is the mine AI provider name (`claude`, `openai`, `gemini`, `openrouter`).

## Write Protocol (Atomic)

All mutations use an atomic write protocol:

1. Marshal plaintext secrets to JSON
2. Encrypt with age scrypt using caller's passphrase
3. Write encrypted bytes to a temp file in the same directory (`.vault-*.tmp`)
4. `chmod 0600` the temp file
5. `fsync` the temp file
6. `os.Rename` temp → final path (atomic on POSIX)

If any step fails, the temp file is removed and the original vault is unchanged.

## Error Handling Policy

| Error | Behavior |
|-------|----------|
| Wrong passphrase | Return `ErrWrongPassphrase`; surface actionable hint; non-zero exit |
| Corrupted/unreadable vault | Return `ErrCorruptedVault`; surface actionable hint; non-zero exit |
| Missing vault file on read | Return `os.ErrNotExist`; callers handle (some treat as empty) |
| Empty key name | Return validation error before any disk I/O |

**No silent fallbacks**. No auto-repair in v1. Users are told explicitly what went wrong and what to do.

## Passphrase Handling

- Passphrase is read from `MINE_VAULT_PASSPHRASE` env var if set.
- Otherwise, prompts interactively via `golang.org/x/term.ReadPassword` (no echo).
- Passphrase is never logged, stored, or written anywhere.
- If neither is available (non-TTY, no env var), commands fail with an explicit actionable error.

## File Permissions

The vault file is always created/updated with mode `0600` (owner read/write only).

## Export/Import

`export` streams the raw encrypted file bytes — the export is itself age-encrypted and safe to store/transfer.

`import` validates the import data by decrypting and parsing it before overwriting the existing vault. If validation fails (wrong passphrase, corrupted data), the existing vault is left intact.

Import **replaces** the vault (no merge). v1 only.

## Concurrency

Each `Vault` instance has a `sync.Mutex` protecting all reads and writes. The underlying POSIX `rename` provides atomic commit semantics for the file. Concurrent processes (separate invocations of `mine vault`) are serialized at the OS level via the rename atomicity — the last writer wins. In practice, vault writes are rare and fast enough that this is not a problem.

## AI Integration

`cmd/ai.go` uses the vault as the canonical API key store:

- `mine ai config --provider <p> --key <k>`: writes `ai.<p>.api_key` to vault
- `mine ai ask/review/commit`: reads `ai.<p>.api_key` from vault (after checking env vars)
- `mine ai config --list`: lists providers by scanning vault keys for `ai.*.api_key` pattern
- `mine ai models`: shows vault-stored providers alongside env-var providers

Env vars always take precedence over vault, enabling zero-config setup for CI/CD.

## v1 Limitations / Future Work

- Passphrase-based only (no hardware key, OS keychain, or age X25519 keys in v1)
- No merge on import (replace only)
- No key rotation (change passphrase by export/re-import)
- No TTL or access logging
- Single vault file (no namespace isolation between vault contents)

## Dependencies

| Package | Purpose |
|---------|---------|
| `filippo.io/age` | Age encryption/decryption |
| `filippo.io/age/armor` | ASCII-armored age format |
| `golang.org/x/term` | Secure password prompt (no echo) |
